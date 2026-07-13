package service_test

import (
	"bytes"
	"context"
	"testing"

	"sms-management/internal/domain"
	"sms-management/internal/repository"
	repomock "sms-management/internal/repository/mock"
	"sms-management/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xuri/excelize/v2"
)

func createExcelBytes(t *testing.T, headers []string, data [][]string) []byte {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Sheet1"

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	for rowIndex, rowData := range data {
		for colIndex, val := range rowData {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			f.SetCellValue(sheetName, cell, val)
		}
	}

	var buf bytes.Buffer
	err := f.Write(&buf)
	assert.NoError(t, err)
	return buf.Bytes()
}

func TestImportServers_ValidFile(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewServerRepository(t)
	outbox := repomock.NewOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	mockTx(repo)

	headers := []string{"Server Name", "IPv4"}
	data := [][]string{
		{"srv-1", "10.0.0.1"},
		{"srv-2", "10.0.0.2"},
	}
	fileBytes := createExcelBytes(t, headers, data)

	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-1", "srv-2"}, []string{"10.0.0.1", "10.0.0.2"}).
		Return([]*domain.Server{}, nil).Once()
	repo.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.Server")).Return(nil).Once()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Once()

	result, err := svc.ImportServers(ctx, fileBytes)

	assert.NoError(t, err)
	assert.Equal(t, int32(2), result.SuccessCount)
	assert.Equal(t, int32(0), result.FailCount)
}

func TestExportServers_Success(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewServerRepository(t)
	outbox := repomock.NewOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	servers := []*domain.Server{
		{ServerID: "id-1", ServerName: "srv-a", IPv4: "1.1.1.1", CurrentStatus: domain.ServerStatusOnline},
	}
	filter := repository.ServerListFilter{Page: 1, PageSize: 100}
	repo.On("Search", ctx, filter).Return(servers, int32(1), nil).Once()

	fileBytes, filename, err := svc.ExportServers(ctx, filter)

	assert.NoError(t, err)
	assert.NotEmpty(t, fileBytes)
	assert.Contains(t, filename, "servers_export")
	assert.Contains(t, filename, ".xlsx")
}
