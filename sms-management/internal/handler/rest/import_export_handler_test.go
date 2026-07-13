package rest

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sms-management/internal/domain"
	"sms-management/internal/repository"
	"sms-management/internal/service"

	"github.com/stretchr/testify/assert"
)

type mockServerService struct {
	importRes  *service.ImportResult
	importErr  error
	exportRes  []byte
	exportName string
	exportErr  error
}

func (m *mockServerService) ImportServers(ctx context.Context, fileBytes []byte) (*service.ImportResult, error) {
	return m.importRes, m.importErr
}

func (m *mockServerService) ExportServers(ctx context.Context, filter repository.ServerListFilter) ([]byte, string, error) {
	return m.exportRes, m.exportName, m.exportErr
}

func (m *mockServerService) CreateServer(ctx context.Context, in service.CreateServerInput) (*domain.Server, error) {
	return nil, nil
}
func (m *mockServerService) UpdateServer(ctx context.Context, id string, in service.UpdateServerInput) (*domain.Server, error) {
	return nil, nil
}
func (m *mockServerService) DeleteServer(ctx context.Context, id string) error { return nil }
func (m *mockServerService) SearchServers(ctx context.Context, filter repository.ServerListFilter) ([]*domain.Server, int32, error) {
	return nil, 0, nil
}
func (m *mockServerService) GetServerDistributionChart(ctx context.Context) (map[string]int32, error) {
	return nil, nil
}

func TestImportExportHandler_HandleImport(t *testing.T) {
	svc := &mockServerService{}
	handler := NewImportExportHandler(svc)

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/import", nil)
		rec := httptest.NewRecorder()
		handler.HandleImport(rec, req)
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("success raw body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/import", strings.NewReader("csv data"))
		req.Header.Set("Content-Type", "text/csv")

		svc.importRes = &service.ImportResult{SuccessCount: 1}
		svc.importErr = nil

		rec := httptest.NewRecorder()
		handler.HandleImport(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("file too large", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/import", strings.NewReader("csv data"))
		req.Header.Set("Content-Type", "text/csv")

		svc.importRes = nil
		svc.importErr = service.ErrFileTooLarge

		rec := httptest.NewRecorder()
		handler.HandleImport(rec, req)
		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})

	t.Run("invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/import", strings.NewReader("csv data"))
		req.Header.Set("Content-Type", "text/csv")

		svc.importRes = nil
		svc.importErr = service.ErrInvalidFormat

		rec := httptest.NewRecorder()
		handler.HandleImport(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/import", strings.NewReader("csv data"))
		req.Header.Set("Content-Type", "text/csv")

		svc.importRes = nil
		svc.importErr = errors.New("db error")

		rec := httptest.NewRecorder()
		handler.HandleImport(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("multipart form data", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		part.Write([]byte("csv data"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/import", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		svc.importRes = &service.ImportResult{SuccessCount: 1}
		svc.importErr = nil

		rec := httptest.NewRecorder()
		handler.HandleImport(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestImportExportHandler_HandleExport(t *testing.T) {
	svc := &mockServerService{}
	handler := NewImportExportHandler(svc)

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/export", nil)
		rec := httptest.NewRecorder()
		handler.HandleExport(rec, req)
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/export?page=1&limit=10&filterStatus=ONLINE&createdFrom=2026-01-01&createdTo=2026-01-02", nil)

		svc.exportRes = []byte("excel data")
		svc.exportName = "export.xlsx"
		svc.exportErr = nil

		rec := httptest.NewRecorder()
		handler.HandleExport(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "excel data", rec.Body.String())
	})

	t.Run("invalid date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/export?createdFrom=invalid", nil)
		rec := httptest.NewRecorder()
		handler.HandleExport(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("service error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/export", nil)

		svc.exportErr = errors.New("db error")

		rec := httptest.NewRecorder()
		handler.HandleExport(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestParseCreatedDateRange(t *testing.T) {
	t.Run("valid full", func(t *testing.T) {
		f, to, err := parseCreatedDateRange("2026-01-01", "2026-01-02")
		assert.NoError(t, err)
		assert.False(t, f.IsZero())
		assert.False(t, to.IsZero())
	})
	t.Run("invalid from", func(t *testing.T) {
		_, _, err := parseCreatedDateRange("2026-13-01", "")
		assert.Error(t, err)
	})
	t.Run("invalid to", func(t *testing.T) {
		_, _, err := parseCreatedDateRange("", "2026-13-01")
		assert.Error(t, err)
	})
	t.Run("from after to", func(t *testing.T) {
		_, _, err := parseCreatedDateRange("2026-02-01", "2026-01-01")
		assert.Error(t, err)
	})
}
