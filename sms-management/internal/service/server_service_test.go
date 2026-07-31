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
	return service.NewServerService(repo, outbox, "dummy_secret", 3)
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

func TestCreateServer_SSHValidations(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := newTestService(repo, outbox)

	repo.On("GetByName", ctx, mock.Anything).Return(nil, nil)
	repo.On("GetByIPv4", ctx, mock.Anything).Return(nil, nil)

	// Invalid Port
	_, err := svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodSSH, ServerName: "s1", IPv4: "1.1.1.1", SSHPort: 0})
	assert.ErrorIs(t, err, service.ErrInvalidSSHPort)

	// Missing User
	_, err = svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodSSH, ServerName: "s1", IPv4: "1.1.1.1", SSHPort: 22, SSHUser: ""})
	assert.ErrorIs(t, err, service.ErrInvalidSSHUser)

	// Missing Key
	_, err = svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodSSH, ServerName: "s1", IPv4: "1.1.1.1", SSHPort: 22, SSHUser: "root", SSHKey: ""})
	assert.ErrorIs(t, err, service.ErrInvalidSSHKey)

	// Invalid Key format
	_, err = svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodSSH, ServerName: "s1", IPv4: "1.1.1.1", SSHPort: 22, SSHUser: "root", SSHKey: "bad key"})
	assert.ErrorIs(t, err, service.ErrInvalidSSHKeyFormat)
}

func TestCreateServer_AgentValidations(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := newTestService(repo, outbox)

	repo.On("GetByName", ctx, mock.Anything).Return(nil, nil)
	repo.On("GetByIPv4", ctx, mock.Anything).Return(nil, nil)

	// Missing Endpoint
	_, err := svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodAgentPull, ServerName: "s1", IPv4: "1.1.1.1", AgentEndpoint: ""})
	assert.ErrorIs(t, err, service.ErrInvalidAgentEndpoint)

	// Invalid Endpoint URL
	_, err = svc.CreateServer(ctx, service.CreateServerInput{HealthCheckMethod: domain.MethodAgentPull, ServerName: "s1", IPv4: "1.1.1.1", AgentEndpoint: "not-a-url"})
	assert.ErrorIs(t, err, service.ErrInvalidAgentEndpointFormat)
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
	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-1"}, []string{"10.0.0.1"}).Return([]*domain.Server{}, nil)
	repo.On("BatchCreate", ctx, mock.Anything).Return(batchErr)

	_, err := svc.ImportServers(ctx, buf.Bytes())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch processing failed")
}

func TestSearchServers_InvalidStatus(t *testing.T) {
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	_, _, err := svc.SearchServers(context.Background(), repository.ServerListFilter{
		Status: "INVALID_STATUS",
	})
	assert.ErrorContains(t, err, "Invalid status filter")
}


func TestDeleteServer_NilServer(t *testing.T) {
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	repo.On("GetByID", mock.Anything, "id-1").Return(nil, nil).Once()
	err := svc.DeleteServer(context.Background(), "id-1")
	assert.ErrorIs(t, err, service.ErrServerNotFound)
}


func TestUpdateServer_NilServer(t *testing.T) {
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	repo.On("GetByID", mock.Anything, "id-1").Return(nil, nil).Once()
	_, err := svc.UpdateServer(context.Background(), "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodICMP, ServerName: "test"})
	assert.ErrorIs(t, err, service.ErrServerNotFound)
}

func TestUpdateServer_MethodSSHEmptyKey(t *testing.T) {
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	existingServer := &domain.Server{
		ServerID:          "id-1",
		ServerName:        "srv-1",
		IPv4:              "1.1.1.1",
		HealthCheckMethod: domain.MethodICMP, // previously ICMP, no SSH key
		SSHKey:            "",
	}

	repo.On("GetByID", mock.Anything, "id-1").Return(existingServer, nil).Once()

	_, err := svc.UpdateServer(context.Background(), "id-1", service.UpdateServerInput{
		ServerName:        "srv-1",
		IPv4:              "1.1.1.1",
		HealthCheckMethod: domain.MethodSSH,
		SSHPort:           22,
		SSHUser:           "root",
		SSHKey:            "", // attempting to leave it empty
	})

	assert.ErrorIs(t, err, service.ErrInvalidSSHKey)
}

func TestUpdateServer_SSHValidations(t *testing.T) {
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	existingServer := &domain.Server{ServerID: "id-1"}
	repo.On("GetByID", mock.Anything, "id-1").Return(existingServer, nil)

	// Invalid Port
	_, err := svc.UpdateServer(context.Background(), "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodSSH, SSHPort: 0})
	assert.ErrorIs(t, err, service.ErrInvalidSSHPort)

	// Missing User
	_, err = svc.UpdateServer(context.Background(), "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodSSH, SSHPort: 22, SSHUser: ""})
	assert.ErrorIs(t, err, service.ErrInvalidSSHUser)

	// Invalid Key format
	_, err = svc.UpdateServer(context.Background(), "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodSSH, SSHPort: 22, SSHUser: "root", SSHKey: "bad key"})
	assert.ErrorIs(t, err, service.ErrInvalidSSHKeyFormat)
}

func TestUpdateServer_AgentValidations(t *testing.T) {
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	existingServer := &domain.Server{ServerID: "id-1"}
	repo.On("GetByID", mock.Anything, "id-1").Return(existingServer, nil)

	// Missing Endpoint
	_, err := svc.UpdateServer(context.Background(), "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodAgentPull, AgentEndpoint: ""})
	assert.ErrorIs(t, err, service.ErrInvalidAgentEndpoint)

	// Invalid Endpoint URL
	_, err = svc.UpdateServer(context.Background(), "id-1", service.UpdateServerInput{HealthCheckMethod: domain.MethodAgentPull, AgentEndpoint: "not-a-url"})
	assert.ErrorIs(t, err, service.ErrInvalidAgentEndpointFormat)
}


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
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

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
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

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
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	fileBytes := make([]byte, 2*1024*1024+1)
	_, err := svc.ImportServers(ctx, fileBytes)
	assert.ErrorIs(t, err, service.ErrFileTooLarge)
}

func TestImportServers_InvalidFormat(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	_, err := svc.ImportServers(ctx, []byte("invalid data"))
	assert.ErrorIs(t, err, service.ErrInvalidFormat)
}

func TestImportServers_MissingColumns(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

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
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)
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
	repo.On("BatchCreate", ctx, mock.Anything).Return(nil)
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
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	filter := repository.ServerListFilter{Page: 1, PageSize: 100}
	repo.On("Search", ctx, filter).Return(nil, int32(0), errors.New("search error")).Once()

	_, _, err := svc.ExportServers(ctx, filter)
	assert.Error(t, err)
}

func TestImportServers_EmptyFile(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

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
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)
	mockTx(repo)

	headers := []string{"Server Name", "IPv4"}
	data := [][]string{
		{"srv-1", "10.0.0.1"},
	}
	fileBytes := createExcelBytes(t, headers, data)

	outboxErr := errors.New("outbox insert failed")
	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-1"}, []string{"10.0.0.1"}).Return([]*domain.Server{}, nil)
	repo.On("BatchCreate", ctx, mock.Anything).Return(nil)
	outbox.On("BatchCreate", ctx, mock.Anything).Return(outboxErr)

	_, err := svc.ImportServers(ctx, fileBytes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create outbox events")
}

func TestImportServers_FindByNamesError(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

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

func TestImportServers_RaceCondition_RetrySuccess(t *testing.T) {
	ctx := context.Background()
	repo := repomock.NewMockServerRepository(t)
	outbox := repomock.NewMockOutboxRepository(t)
	svc := service.NewServerService(repo, outbox, "dummy_secret", 3)

	headers := []string{"Server Name", "IPv4"}
	data := [][]string{
		{"srv-1", "10.0.0.1"},
	}
	fileBytes := createExcelBytes(t, headers, data)

	// Attempt 1: Find returns no duplicates
	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-1"}, []string{"10.0.0.1"}).Return([]*domain.Server{}, nil).Once()
	
	// Attempt 1: ExecuteInTx fails with unique constraint violation (simulating BatchCreate failing inside)
	dbErr := errors.New("duplicate key value violates unique constraint")
	repo.On("ExecuteInTx", ctx, mock.Anything).Return(dbErr).Once()
	
	// Attempt 2 (Retry): Find returns the duplicated server
	dupServer := &domain.Server{ServerName: "srv-1", IPv4: "10.0.0.1"}
	repo.On("FindByNamesOrIPv4s", ctx, []string{"srv-1"}, []string{"10.0.0.1"}).Return([]*domain.Server{dupServer}, nil).Once()
	
	// ExecuteInTx is skipped because validServers is empty

	result, err := svc.ImportServers(ctx, fileBytes)
	assert.NoError(t, err)
	
	assert.Equal(t, int32(0), result.SuccessCount)
	assert.Equal(t, int32(1), result.FailCount)
	assert.Contains(t, result.FailedServers[0], "srv-1")
}
