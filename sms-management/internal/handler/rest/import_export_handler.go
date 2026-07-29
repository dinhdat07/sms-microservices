package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sms-management/internal/repository"
	"sms-management/internal/service"
)

// ImportExportHandler handles file import/export via REST, bypassing gRPC-gateway.
type ImportExportHandler struct {
	svc     service.ServerService
	maxSize int64
}

func NewImportExportHandler(svc service.ServerService) *ImportExportHandler {
	return &ImportExportHandler{
		svc:     svc,
		maxSize: 2 << 20, // 2MB
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[rest] failed to encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"code":    status,
		"message": message,
	})
}


func (h *ImportExportHandler) HandleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxSize)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body.")
		return
	}

	// Try multipart; fall back to raw body (for Bruno CLI compatibility)
	fileBytes := h.extractFileFromMultipart(r.Header.Get("Content-Type"), raw)
	if fileBytes == nil {
		fileBytes = raw
	}

	result, err := h.svc.ImportServers(r.Context(), fileBytes)
	if err != nil {
		if errors.Is(err, service.ErrFileTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidFormat) || errors.Is(err, service.ErrEmptyFile) ||
			errors.Is(err, service.ErrNoSheets) || errors.Is(err, service.ErrMissingCols) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"successCount":      result.SuccessCount,
		"successfulServers": result.SuccessfulServers,
		"failCount":         result.FailCount,
		"failedServers":     result.FailedServers,
	})
}

func (h *ImportExportHandler) extractFileFromMultipart(contentType string, raw []byte) []byte {
	boundary := extractBoundary(contentType)
	if boundary == "" || len(raw) < 2 || string(raw[:2]) != "--" {
		return nil
	}
	mr := multipart.NewReader(bytes.NewReader(raw), boundary)
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		if part.FormName() == "file" {
			data, _ := io.ReadAll(part)
			return data
		}
	}
	return nil
}

// extractBoundary returns the boundary value from a Content-Type header.
func extractBoundary(contentType string) string {
	for _, part := range strings.Split(contentType, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "boundary=") {
			return strings.TrimPrefix(part, "boundary=")
		}
	}
	return ""
}

func (h *ImportExportHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 100
	}

	filter := repository.ServerListFilter{
		Page:          page,
		PageSize:      limit,
		Status:        q.Get("filterStatus"),
		Name:          q.Get("filterName"),
		SortBy:        q.Get("sortBy"),
		SortDirection: q.Get("sortDirection"),
	}
	createdFrom, createdTo, err := parseCreatedDateRange(q.Get("createdFrom"), q.Get("createdTo"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	filter.CreatedFrom = createdFrom
	filter.CreatedTo = createdTo

	fileBytes, filename, err := h.svc.ExportServers(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(fileBytes)
}

func parseCreatedDateRange(createdFrom, createdTo string) (time.Time, time.Time, error) {
	var from time.Time
	var to time.Time
	var err error
	if createdFrom != "" {
		from, err = time.Parse("2006-01-02", createdFrom)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("Created From must use YYYY-MM-DD format.")
		}
	}
	if createdTo != "" {
		to, err = time.Parse("2006-01-02", createdTo)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("Created To must use YYYY-MM-DD format.")
		}
		to = to.AddDate(0, 0, 1)
	}
	if !from.IsZero() && !to.IsZero() && !from.Before(to) {
		return time.Time{}, time.Time{}, errors.New("Created From must be before or equal to Created To.")
	}
	return from, to, nil
}
