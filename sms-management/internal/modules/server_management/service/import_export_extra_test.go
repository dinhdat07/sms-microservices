package service_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"sms-management/internal/modules/server_management/domain"
	repomock "sms-management/internal/modules/server_management/repository/mock"
	"sms-management/internal/modules/server_management/service"
	cachemock "sms-management/internal/modules/server_management/service/mock"

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

func TestImportServers_FindByNamesOrIPv4s_Error(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	svc := service.NewServerService(repo, nil)

	headers := []string{"Server Name", "IPv4"}
	data := [][]string{{"srv-err", "10.0.0.1"}}
	fileBytes := createExcelBytes(t, headers, data)

	dbErr := errors.New("db error")
	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-err"}, []string{"10.0.0.1"}).Return(nil, dbErr).Once()

	res, err := svc.ImportServers(ctx, fileBytes)

	assert.ErrorIs(t, err, dbErr)
	assert.Nil(t, res)
}

func TestImportServers_InvalidIP(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	svc := service.NewServerService(repo, nil)

	headers := []string{"Server Name", "IPv4"}
	data := [][]string{{"srv-invalid", "999.999.999"}}
	fileBytes := createExcelBytes(t, headers, data)

	res, err := svc.ImportServers(ctx, fileBytes)

	assert.NoError(t, err)
	assert.Equal(t, int32(0), res.SuccessCount)
	assert.Equal(t, int32(1), res.FailCount)
	assert.Contains(t, res.FailedServers[0], "Invalid Format")
}

func TestImportServers_CacheUpsert_Error(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	cache := cachemock.NewCacheManager(t)
	svc := service.NewServerService(repo, cache)

	headers := []string{"Server Name", "IPv4"}
	data := [][]string{{"srv-cache-err", "10.0.0.2"}}
	fileBytes := createExcelBytes(t, headers, data)

	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-cache-err"}, []string{"10.0.0.2"}).Return([]*domain.Server{}, nil).Once()
	repo.On("BatchCreate", ctx, mock.Anything).Return(nil).Once()
	
	cacheErr := errors.New("redis error")
	cache.On("BatchUpsert", ctx, mock.Anything).Return(cacheErr).Once()

	res, err := svc.ImportServers(ctx, fileBytes)

	assert.NoError(t, err)
	assert.Equal(t, int32(1), res.SuccessCount) // still success even if cache fails
}

func TestImportServers_MultipleBatches(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	svc := service.NewServerService(repo, nil)

	headers := []string{"Server Name", "IPv4"}
	var data [][]string
	for i := 0; i < 105; i++ {
		data = append(data, []string{fmt.Sprintf("srv-%d", i), fmt.Sprintf("10.0.0.%d", i%255)})
	}
	fileBytes := createExcelBytes(t, headers, data)

	repo.On("FindByNamesOrIPv4s", ctx, mock.Anything, mock.Anything).Return([]*domain.Server{}, nil).Twice()
	repo.On("BatchCreate", ctx, mock.Anything).Return(nil).Twice()

	res, err := svc.ImportServers(ctx, fileBytes)

	assert.NoError(t, err)
	assert.Equal(t, int32(105), res.SuccessCount)
}
