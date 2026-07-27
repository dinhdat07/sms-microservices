package service_test

import (
	"bytes"
	"context"
	"errors"
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
		_ = f.SetCellValue(sheetName, cell, header)
	}

	for rowIndex, rowData := range data {
		for colIndex, val := range rowData {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}

	var buf bytes.Buffer
	err := f.Write(&buf)
	assert.NoError(t, err)
	return buf.Bytes()
}

func TestImportServers_ValidFile(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
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
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
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

func TestImportServers_FileTooLarge(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	fileBytes := make([]byte, 2*1024*1024+1)
	_, err := svc.ImportServers(ctx, fileBytes)
	assert.ErrorIs(t, err, service.ErrFileTooLarge)
}

func TestImportServers_InvalidFormat(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	_, err := svc.ImportServers(ctx, []byte("invalid data"))
	assert.ErrorIs(t, err, service.ErrInvalidFormat)
}

func TestImportServers_MissingColumns(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	headers := []string{"Unknown"}
	data := [][]string{{"test"}}
	fileBytes := createExcelBytes(t, headers, data)

	_, err := svc.ImportServers(ctx, fileBytes)
	assert.ErrorIs(t, err, service.ErrMissingCols)
}

func TestImportServers_Duplicates(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)
	mockTx(repo)

	headers := []string{"Server Name", "IPv4"}
	data := [][]string{
		{"srv-1", "10.0.0.1"},
		{"srv-1", "10.0.0.2"}, // Duplicate name within batch
		{"srv-3", "10.0.0.1"}, // Duplicate IP within batch
		{"srv-4", "10.0.0.4"},
	}
	fileBytes := createExcelBytes(t, headers, data)

	existingServers := []*domain.Server{
		{ServerName: "srv-4", IPv4: "10.0.0.4"},
	}

	repo.On("FindByNamesOrIPv4s", ctx, mock.Anything, mock.Anything).Return(existingServers, nil).Once()
	repo.On("BatchCreate", ctx, mock.Anything).Return(nil).Once()
	outbox.On("BatchCreate", ctx, mock.Anything).Return(nil).Once()

	result, err := svc.ImportServers(ctx, fileBytes)
	assert.NoError(t, err)
	// srv-1 (valid), srv-1 (dup name -> fail), srv-3 (dup IP -> fail), srv-4 (existing -> fail)
	assert.Equal(t, int32(1), result.SuccessCount)
	assert.Equal(t, int32(3), result.FailCount)
}

func TestExportServers_Error(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	filter := repository.ServerListFilter{Page: 1, PageSize: 100}
	repo.On("Search", ctx, filter).Return(nil, int32(0), errors.New("search error")).Once()

	_, _, err := svc.ExportServers(ctx, filter)
	assert.Error(t, err)
}

func TestImportServers_EmptyFile(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	headers := []string{}
	data := [][]string{}
	fileBytes := createExcelBytes(t, headers, data)

	_, err := svc.ImportServers(ctx, fileBytes)
	assert.ErrorIs(t, err, service.ErrEmptyFile)
}

func TestImportServers_OutboxError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)
	mockTx(repo)

	headers := []string{"Server Name", "IPv4"}
	data := [][]string{
		{"srv-1", "10.0.0.1"},
	}
	fileBytes := createExcelBytes(t, headers, data)

	outboxErr := errors.New("outbox insert failed")
	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-1"}, []string{"10.0.0.1"}).Return([]*domain.Server{}, nil).Once()
	repo.On("BatchCreate", ctx, mock.Anything).Return(nil).Once()
	outbox.On("BatchCreate", ctx, mock.Anything).Return(outboxErr).Once()

	_, err := svc.ImportServers(ctx, fileBytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create outbox events")
}

func TestImportServers_FindByNamesError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	headers := []string{"Server Name", "IPv4"}
	data := [][]string{
		{"srv-1", "10.0.0.1"},
	}
	fileBytes := createExcelBytes(t, headers, data)

	dbErr := errors.New("db error")
	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-1"}, []string{"10.0.0.1"}).Return(nil, dbErr).Once()

	_, err := svc.ImportServers(ctx, fileBytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check existing servers")
}
