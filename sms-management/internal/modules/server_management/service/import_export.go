package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"sms-management/internal/shared/logger"
	"strings"
	"time"

	"sms-management/internal/infrastructure/redis"
	"sms-management/internal/modules/server_management/domain"
	"sms-management/internal/modules/server_management/repository"

	"github.com/xuri/excelize/v2"
)

const maxFileSize = 2 * 1024 * 1024 // 2MB used to limit the size of the imported excel file
const batchSize = 100               // 100 servers per batch

var (
	ErrFileTooLarge  = errors.New("file size exceeds 2MB limit")
	ErrInvalidFormat = errors.New("invalid excel file format")
	ErrNoSheets      = errors.New("no sheets found in the excel file")
	ErrEmptyFile     = errors.New("empty excel file")
	ErrMissingCols   = errors.New("missing required columns: 'Server Name' or 'IPv4'")
)

func (s *serverService) ImportServers(ctx context.Context, fileBytes []byte) (*ImportResult, error) {
	if len(fileBytes) > maxFileSize {
		return nil, ErrFileTooLarge
	}

	f, err := excelize.OpenReader(bytes.NewReader(fileBytes))
	if err != nil {
		return nil, ErrInvalidFormat
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, ErrNoSheets
	}

	rows, err := f.Rows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	// Read header
	if !rows.Next() {
		return nil, ErrEmptyFile
	}
	header, _ := rows.Columns()

	nameIdx, ipIdx := -1, -1
	for i, h := range header {
		hLower := strings.TrimSpace(strings.ToLower(h))
		switch hLower {
		case "server name", "server_name", "name":
			nameIdx = i
		case "ipv4", "ip":
			ipIdx = i
		}
	}

	if nameIdx == -1 || ipIdx == -1 {
		return nil, ErrMissingCols
	}

	result := &ImportResult{}
	var batch []*domain.Server

	processBatch := func() error {
		if len(batch) == 0 {
			return nil
		}

		names := make([]string, 0, len(batch))
		ips := make([]string, 0, len(batch))
		for _, srv := range batch {
			names = append(names, srv.ServerName)
			ips = append(ips, srv.IPv4)
		}

		existingServers, err := s.repo.FindByNamesOrIPv4s(ctx, names, ips)
		if err != nil {
			return fmt.Errorf("failed to check existing servers: %w", err)
		}

		existingNameMap := make(map[string]bool)
		existingIPMap := make(map[string]bool)
		for _, es := range existingServers {
			existingNameMap[es.ServerName] = true
			existingIPMap[es.IPv4] = true
		}

		var validServers []*domain.Server
		for _, srv := range batch {
			if existingNameMap[srv.ServerName] || existingIPMap[srv.IPv4] {
				result.FailCount++
				result.FailedServers = append(result.FailedServers, fmt.Sprintf("%s (%s)", srv.ServerName, srv.IPv4))
			} else {
				validServers = append(validServers, srv)
				result.SuccessCount++
				result.SuccessfulServers = append(result.SuccessfulServers, srv.ServerName)

				// Prevent duplicates within the same batch from bypassing the check
				existingNameMap[srv.ServerName] = true
				existingIPMap[srv.IPv4] = true
			}
		}

		if len(validServers) > 0 {
			if err := s.repo.BatchCreate(ctx, validServers); err != nil {
				return fmt.Errorf("batch create failed: %w", err)
			}

			// Dual-Write to Redis using Pipeline
			if s.cache != nil {
				var cacheItems []redis.CacheUpsertItem
				for _, srv := range validServers {
					cacheItems = append(cacheItems, redis.CacheUpsertItem{
						ID:         srv.ServerID,
						IPv4:       srv.IPv4,
						Status:     string(srv.CurrentStatus),
						RetryCount: 0,
					})
				}
				if err := s.cache.BatchUpsert(ctx, cacheItems); err != nil {
					logger.Log.Sugar().Errorf("[WARNING] DB Import succeeded but Redis sync failed: %v", err)
				}
			}
		}

		batch = batch[:0] // clear batch
		return nil
	}

	for rows.Next() {
		cols, _ := rows.Columns()

		var name, ipv4 string
		if nameIdx < len(cols) {
			name = strings.TrimSpace(cols[nameIdx])
		}
		if ipIdx < len(cols) {
			ipv4 = strings.TrimSpace(cols[ipIdx])
		}

		if name == "" && ipv4 == "" {
			continue // skip completely empty rows
		}

		// simple validation before adding to batch
		ip := net.ParseIP(ipv4)
		if name == "" || ipv4 == "" || len(name) > 255 || ip == nil || ip.To4() == nil {
			result.FailCount++
			result.FailedServers = append(result.FailedServers, fmt.Sprintf("%s (%s) - Invalid Format", name, ipv4))
			continue
		}

		batch = append(batch, &domain.Server{
			ServerName:    name,
			IPv4:          ipv4,
			CurrentStatus: domain.ServerStatusOnline,
		})

		if len(batch) >= batchSize {
			if err := processBatch(); err != nil {
				return nil, err
			}
		}
	}

	// Process remaining
	if err := processBatch(); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *serverService) ExportServers(ctx context.Context, filter repository.ServerListFilter) ([]byte, string, error) {
	// Query servers from Postgres
	servers, _, err := s.repo.Search(ctx, filter)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch servers for export: %w", err)
	}

	buf, err := generateServerExcel(servers)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("servers_export_%s.xlsx", time.Now().Format("20060102_150405"))
	return buf, filename, nil
}

func generateServerExcel(servers []*domain.Server) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Set column widths for better balance
	f.SetColWidth("Sheet1", "A", "A", 30) // Server Name
	f.SetColWidth("Sheet1", "B", "B", 20) // IPv4
	f.SetColWidth("Sheet1", "C", "C", 15) // Status
	f.SetColWidth("Sheet1", "D", "E", 25) // Dates

	sw, err := f.NewStreamWriter("Sheet1")
	if err != nil {
		return nil, fmt.Errorf("failed to init stream writer: %w", err)
	}

	// Create header style
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#0b4884"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create style: %w", err)
	}

	// Write header
	headers := []interface{}{
		excelize.Cell{Value: "Server Name", StyleID: headerStyle},
		excelize.Cell{Value: "IPv4 Address", StyleID: headerStyle},
		excelize.Cell{Value: "Status", StyleID: headerStyle},
		excelize.Cell{Value: "Created At", StyleID: headerStyle},
		excelize.Cell{Value: "Last Updated", StyleID: headerStyle},
	}
	if err := sw.SetRow("A1", headers); err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	// Write data
	for i, srv := range servers {
		rowIdx := i + 2 // Start from row 2
		cell, _ := excelize.CoordinatesToCellName(1, rowIdx)

		row := []interface{}{
			srv.ServerName,
			srv.IPv4,
			string(srv.CurrentStatus),
			srv.CreatedAt.Format("2006-01-02 15:04:05"),
			srv.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := sw.SetRow(cell, row); err != nil {
			return nil, fmt.Errorf("failed to write row %d: %w", rowIdx, err)
		}
	}

	if err := sw.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush stream: %w", err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to write excel to buffer: %w", err)
	}

	return buf.Bytes(), nil
}
