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

func newTestService(repo repository.ServerRepository, outbox repository.OutboxRepository) service.ServerService {
	return service.NewServerService(repo, outbox)
}

func mockTx(repo *repomock.MockServerRepository) {
	repo.On("ExecuteInTx", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Maybe()
}

func TestCreateServer_Success(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	repo.On("GetByName", ctx, "test-server").Return(nil, nil).Once()
	repo.On("GetByIPv4", ctx, "1.2.3.4").Return(nil, nil).Once()
	repo.On("Create", ctx, mock.AnythingOfType("*domain.Server")).Return(nil).Once()

	server, err := svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "test-server", IPv4: "1.2.3.4"})

	assert.NoError(t, err)
	assert.Equal(t, "test-server", server.ServerName)
	assert.Equal(t, "1.2.3.4", server.IPv4)
}

func TestCreateServer_NameExists(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "existing-id", ServerName: "test-server", IPv4: "5.6.7.8"}
	repo.On("GetByName", ctx, "test-server").Return(existing, nil).Once()

	_, err := svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "test-server", IPv4: "1.2.3.4"})

	assert.ErrorIs(t, err, service.ErrNameExists)
}

func TestCreateServer_IPv4Exists(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "existing-id", ServerName: "other", IPv4: "1.2.3.4"}
	repo.On("GetByName", ctx, "test-server").Return(nil, nil).Once()
	repo.On("GetByIPv4", ctx, "1.2.3.4").Return(existing, nil).Once()

	_, err := svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "test-server", IPv4: "1.2.3.4"})

	assert.ErrorIs(t, err, service.ErrIPv4Exists)
}

func TestCreateServer_DBError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	dbErr := errors.New("db connection lost")
	repo.On("GetByName", ctx, "test-server").Return(nil, nil).Once()
	repo.On("GetByIPv4", ctx, "1.2.3.4").Return(nil, nil).Once()
	repo.On("Create", ctx, mock.AnythingOfType("*domain.Server")).Return(dbErr).Once()

	_, err := svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "test-server", IPv4: "1.2.3.4"})

	assert.ErrorIs(t, err, dbErr)
}

func TestUpdateServer_Success(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "id-1", ServerName: "old-name", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("GetByName", ctx, "new-name").Return(nil, nil).Once()
	repo.On("GetByIPv4", ctx, "2.2.2.2").Return(nil, nil).Once()
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Server")).Return(nil).Once()

	server, err := svc.UpdateServer(ctx, "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "new-name", IPv4: "2.2.2.2"})

	assert.NoError(t, err)
	assert.Equal(t, "new-name", server.ServerName)
	assert.Equal(t, "2.2.2.2", server.IPv4)
}

func TestUpdateServer_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	repo.On("GetByID", ctx, "id-nonexistent").Return(nil, repository.ErrNotFound).Once()

	_, err := svc.UpdateServer(ctx, "id-nonexistent", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "new-name", IPv4: "2.2.2.2"})

	assert.ErrorIs(t, err, service.ErrServerNotFound)
}

func TestUpdateServer_NameConflict(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "id-1", ServerName: "old-name", IPv4: "1.1.1.1"}
	conflict := &domain.Server{ServerID: "id-2", ServerName: "new-name", IPv4: "3.3.3.3"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("GetByName", ctx, "new-name").Return(conflict, nil).Once()

	_, err := svc.UpdateServer(ctx, "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "new-name", IPv4: "2.2.2.2"})

	assert.ErrorIs(t, err, service.ErrNameExists)
}

func TestUpdateServer_IPv4Conflict(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "id-1", ServerName: "old-name", IPv4: "1.1.1.1"}
	conflict := &domain.Server{ServerID: "id-3", ServerName: "other", IPv4: "2.2.2.2"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("GetByName", ctx, "new-name").Return(nil, nil).Once()
	repo.On("GetByIPv4", ctx, "2.2.2.2").Return(conflict, nil).Once()

	_, err := svc.UpdateServer(ctx, "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "new-name", IPv4: "2.2.2.2"})

	assert.ErrorIs(t, err, service.ErrIPv4Exists)
}

func TestDeleteServer_Success(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "id-1", ServerName: "srv", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("Delete", ctx, "id-1").Return(nil).Once()

	err := svc.DeleteServer(ctx, "id-1")
	assert.NoError(t, err)
}

func TestDeleteServer_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	repo.On("GetByID", ctx, "id-nonexistent").Return(nil, repository.ErrNotFound).Once()

	err := svc.DeleteServer(ctx, "id-nonexistent")
	assert.ErrorIs(t, err, service.ErrServerNotFound)
}

func TestDeleteServer_DBError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	dbErr := errors.New("db crash")
	existing := &domain.Server{ServerID: "id-1", ServerName: "srv", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("Delete", ctx, "id-1").Return(dbErr).Once()

	err := svc.DeleteServer(ctx, "id-1")
	assert.ErrorIs(t, err, dbErr)
}

func TestSearchServers_WithFilters(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	servers := []*domain.Server{
		{ServerID: "id-1", ServerName: "srv-a", IPv4: "1.1.1.1", CurrentStatus: domain.ServerStatusOnline},
		{ServerID: "id-2", ServerName: "srv-b", IPv4: "2.2.2.2", CurrentStatus: domain.ServerStatusOffline},
	}
	filter := repository.ServerListFilter{Page: 1, PageSize: 20, Status: "ONLINE", Name: "srv"}
	repo.On("Search", ctx, filter).Return(servers, int32(2), nil).Once()

	results, total, err := svc.SearchServers(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, int32(2), total)
	assert.Len(t, results, 2)
}

func TestSearchServers_Empty(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	filter := repository.ServerListFilter{Page: 1, PageSize: 20}
	repo.On("Search", ctx, filter).Return([]*domain.Server{}, int32(0), nil).Once()

	results, total, err := svc.SearchServers(ctx, filter)

	assert.NoError(t, err)
	assert.Equal(t, int32(0), total)
	assert.Empty(t, results)
}

func TestSearchServers_DBError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	dbErr := errors.New("query failed")
	filter := repository.ServerListFilter{Page: 1, PageSize: 20}
	repo.On("Search", ctx, filter).Return(nil, int32(0), dbErr).Once()

	_, _, err := svc.SearchServers(ctx, filter)
	assert.ErrorIs(t, err, dbErr)
}

// --- Phase 2: Critical Missing Test Cases ---

// Case 1: CreateServer - GetByName DB error
func TestCreateServer_GetByNameError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	dbErr := errors.New("db connection lost")
	repo.On("GetByName", ctx, "test").Return(nil, dbErr).Once()

	_, err := svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "test", IPv4: "1.2.3.4"})
	assert.ErrorIs(t, err, dbErr)
}

// Case 2: CreateServer - GetByIPv4 DB error
func TestCreateServer_GetByIPv4Error(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	dbErr := errors.New("query timeout")
	repo.On("GetByName", ctx, "test").Return(nil, nil).Once()
	repo.On("GetByIPv4", ctx, "1.2.3.4").Return(nil, dbErr).Once()

	_, err := svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "test", IPv4: "1.2.3.4"})
	assert.ErrorIs(t, err, dbErr)
}

// Case 3: UpdateServer - Same name, different IP (skip GetByName)
func TestUpdateServer_SameNameDiffIP(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "id-1", ServerName: "srv", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	// GetByName should NOT be called (name unchanged)
	repo.On("GetByIPv4", ctx, "2.2.2.2").Return(nil, nil).Once()
	repo.On("Update", ctx, mock.MatchedBy(func(s *domain.Server) bool {
		return s.ServerName == "srv" && s.IPv4 == "2.2.2.2"
	})).Return(nil).Once()

	server, err := svc.UpdateServer(ctx, "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "srv", IPv4: "2.2.2.2"})

	assert.NoError(t, err)
	assert.Equal(t, "srv", server.ServerName)
	assert.Equal(t, "2.2.2.2", server.IPv4)
	repo.AssertNotCalled(t, "GetByName")
}

// Case 4: UpdateServer - Same IP, different name (skip GetByIPv4)
func TestUpdateServer_SameIPDiffName(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "id-1", ServerName: "old", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("GetByName", ctx, "new-name").Return(nil, nil).Once()
	// GetByIPv4 should NOT be called (IP unchanged)
	repo.On("Update", ctx, mock.MatchedBy(func(s *domain.Server) bool {
		return s.ServerName == "new-name" && s.IPv4 == "1.1.1.1"
	})).Return(nil).Once()

	_, err := svc.UpdateServer(ctx, "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "new-name", IPv4: "1.1.1.1"})

	assert.NoError(t, err)
	repo.AssertNotCalled(t, "GetByIPv4")
}

// Case 5: UpdateServer - Identical values (skip all conflict checks)
func TestUpdateServer_IdenticalValues(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "id-1", ServerName: "srv", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("Update", ctx, mock.MatchedBy(func(s *domain.Server) bool {
		return s.ServerID == "id-1" && s.ServerName == "srv" && s.IPv4 == "1.1.1.1"
	})).Return(nil).Once()

	_, err := svc.UpdateServer(ctx, "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "srv", IPv4: "1.1.1.1"})

	assert.NoError(t, err)
	repo.AssertNotCalled(t, "GetByName")
	repo.AssertNotCalled(t, "GetByIPv4")
}

// Case 6: UpdateServer - GetByName returns error during conflict check
func TestUpdateServer_GetByNameError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	dbErr := errors.New("db timeout")
	existing := &domain.Server{ServerID: "id-1", ServerName: "old", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("GetByName", ctx, "new-name").Return(nil, dbErr).Once()

	_, err := svc.UpdateServer(ctx, "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "new-name", IPv4: "1.1.1.1"})
	assert.ErrorIs(t, err, dbErr)
}

// Case 7: UpdateServer - GetByIPv4 returns error during conflict check
func TestUpdateServer_GetByIPv4Error(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	dbErr := errors.New("db timeout")
	existing := &domain.Server{ServerID: "id-1", ServerName: "srv", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("GetByIPv4", ctx, "2.2.2.2").Return(nil, dbErr).Once()

	_, err := svc.UpdateServer(ctx, "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "srv", IPv4: "2.2.2.2"})
	assert.ErrorIs(t, err, dbErr)
}

// Case 8: UpdateServer - repo.Update returns error
func TestUpdateServer_UpdateError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	dbErr := errors.New("update failed")
	existing := &domain.Server{ServerID: "id-1", ServerName: "old", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("GetByName", ctx, "new-name").Return(nil, nil).Once()
	repo.On("GetByIPv4", ctx, "2.2.2.2").Return(nil, nil).Once()
	repo.On("Update", ctx, mock.Anything).Return(dbErr).Once()

	_, err := svc.UpdateServer(ctx, "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "new-name", IPv4: "2.2.2.2"})
	assert.ErrorIs(t, err, dbErr)
}

// Case 9: DeleteServer - repo.Delete returns ErrNotFound (maps to service.ErrServerNotFound)
func TestDeleteServer_DeleteReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	existing := &domain.Server{ServerID: "id-1", ServerName: "srv", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("Delete", ctx, "id-1").Return(repository.ErrNotFound).Once()

	err := svc.DeleteServer(ctx, "id-1")
	assert.ErrorIs(t, err, service.ErrServerNotFound)
}

// Case 10: DeleteServer - repo.Delete returns generic error
func TestDeleteServer_DeleteError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	dbErr := errors.New("delete failed")
	existing := &domain.Server{ServerID: "id-1", ServerName: "srv", IPv4: "1.1.1.1"}
	repo.On("GetByID", ctx, "id-1").Return(existing, nil).Once()
	repo.On("Delete", ctx, "id-1").Return(dbErr).Once()

	err := svc.DeleteServer(ctx, "id-1")
	assert.ErrorIs(t, err, dbErr)
}

// Case 11: SearchServers - Page=0 defaults to 1
func TestSearchServers_PageZeroDefaultsToOne(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	// The service should normalize Page=0 to Page=1
	filter := repository.ServerListFilter{Page: 0, PageSize: 20}
	repo.On("Search", ctx, repository.ServerListFilter{Page: 1, PageSize: 20}).Return([]*domain.Server{}, int32(0), nil).Once()

	results, total, err := svc.SearchServers(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), total)
	assert.Empty(t, results)
}

// Case 12: SearchServers - PageSize=0 defaults to 20
func TestSearchServers_PageSizeZeroDefaults(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	filter := repository.ServerListFilter{Page: 1, PageSize: 0}
	repo.On("Search", ctx, repository.ServerListFilter{Page: 1, PageSize: 20}).Return([]*domain.Server{}, int32(0), nil).Once()

	_, _, err := svc.SearchServers(ctx, filter)
	assert.NoError(t, err)
}

// Case 13: SearchServers - PageSize=101 clamped to 20
func TestSearchServers_PageSizeExceedsMax(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	filter := repository.ServerListFilter{Page: 1, PageSize: 101}
	repo.On("Search", ctx, repository.ServerListFilter{Page: 1, PageSize: 20}).Return([]*domain.Server{}, int32(0), nil).Once()

	_, _, err := svc.SearchServers(ctx, filter)
	assert.NoError(t, err)
}

// Case 14: ImportServers - BatchCreate returns error
func TestImportServers_BatchCreateError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	outbox.On("Create", ctx, mock.AnythingOfType("*domain.OutboxEvent")).Return(nil).Maybe()
	outbox.On("BatchCreate", ctx, mock.AnythingOfType("[]*domain.OutboxEvent")).Return(nil).Maybe()
	svc := newTestService(repo, outbox)
	mockTx(repo)

	f := excelize.NewFile()
	_ = f.SetCellValue("Sheet1", "A1", "Server Name")
	_ = f.SetCellValue("Sheet1", "B1", "IPv4")
	_ = f.SetCellValue("Sheet1", "A2", "srv-1")
	_ = f.SetCellValue("Sheet1", "B2", "10.0.0.1")

	buf := new(bytes.Buffer)
	_ = f.Write(buf)

	batchErr := errors.New("batch insert failed")
	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-1"}, []string{"10.0.0.1"}).Return([]*domain.Server{}, nil).Once()
	repo.On("BatchCreate", ctx, mock.Anything).Return(batchErr).Once()

	_, err := svc.ImportServers(ctx, buf.Bytes())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch create failed")
}

func TestSearchServers_InvalidStatus(t *testing.T) {
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	_, _, err := svc.SearchServers(context.Background(), repository.ServerListFilter{
		Status: "INVALID_STATUS",
	})
	assert.ErrorContains(t, err, "Invalid status filter")
}


func TestDeleteServer_NilServer(t *testing.T) {
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	repo.On("GetByID", mock.Anything, "id-1").Return(nil, nil).Once()
	err := svc.DeleteServer(context.Background(), "id-1")
	assert.ErrorIs(t, err, service.ErrServerNotFound)
}


func TestUpdateServer_NilServer(t *testing.T) {
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox)

	repo.On("GetByID", mock.Anything, "id-1").Return(nil, nil).Once()
	_, err := svc.UpdateServer(context.Background(), "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "test"})
	assert.ErrorIs(t, err, service.ErrServerNotFound)
}

