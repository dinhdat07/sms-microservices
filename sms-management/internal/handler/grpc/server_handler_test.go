package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	server_managementv1 "sms-management/gen/go/server_management/v1"
	"sms-management/internal/domain"
	"sms-management/internal/repository"
	"sms-management/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockSvc struct {
	mock.Mock
}

func (m *mockSvc) CreateServer(ctx context.Context, input service.CreateServerInput) (*domain.Server, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Server), args.Error(1)
}

func (m *mockSvc) UpdateServer(ctx context.Context, id string, input service.UpdateServerInput) (*domain.Server, error) {
	args := m.Called(ctx, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Server), args.Error(1)
}

func (m *mockSvc) DeleteServer(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockSvc) SearchServers(ctx context.Context, filter repository.ServerListFilter) ([]*domain.Server, int32, error) {
	args := m.Called(ctx, filter)
	var servers []*domain.Server
	if args.Get(0) != nil {
		servers = args.Get(0).([]*domain.Server)
	}
	return servers, args.Get(1).(int32), args.Error(2)
}

func (m *mockSvc) ImportServers(ctx context.Context, fileBytes []byte) (*service.ImportResult, error) {
	args := m.Called(ctx, fileBytes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ImportResult), args.Error(1)
}

func (m *mockSvc) ExportServers(ctx context.Context, filter repository.ServerListFilter) ([]byte, string, error) {
	args := m.Called(ctx, filter)
	var b []byte
	if args.Get(0) != nil {
		b = args.Get(0).([]byte)
	}
	return b, args.String(1), args.Error(2)
}

func (m *mockSvc) GenerateAgentToken(ctx context.Context, serverID string) (string, error) {
	args := m.Called(ctx, serverID)
	return args.String(0), args.Error(1)
}

func TestHandler_CreateServer_Success(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("CreateServer", mock.Anything, service.CreateServerInput{ServerName: "srv", IPv4: "1.2.3.4", HealthCheckMethod: "ICMP"}).
		Return(&domain.Server{ServerID: "uuid-1", ServerName: "srv", IPv4: "1.2.3.4", CurrentStatus: domain.ServerStatusOnline}, nil).Once()

	resp, err := h.CreateServer(context.Background(), &server_managementv1.CreateServerRequest{
		ServerName: "srv", Ipv4: "1.2.3.4",
	})

	assert.NoError(t, err)
	assert.Equal(t, "uuid-1", resp.Server.ServerId)
	assert.Equal(t, "ONLINE", resp.Server.CurrentStatus)
}

func TestHandler_CreateServer_AlreadyExists(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("CreateServer", mock.Anything, mock.Anything).Return(nil, service.ErrNameExists).Once()

	_, err := h.CreateServer(context.Background(), &server_managementv1.CreateServerRequest{
		ServerName: "dup", Ipv4: "1.2.3.4",
	})

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
}

func TestHandler_DeleteServer_NotFound(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("DeleteServer", mock.Anything, "id-missing").Return(service.ErrServerNotFound).Once()

	_, err := h.DeleteServer(context.Background(), &server_managementv1.DeleteServerRequest{ServerId: "id-missing"})

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestHandler_DeleteServer_InternalError(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("DeleteServer", mock.Anything, "id-1").Return(errors.New("db crash")).Once()

	_, err := h.DeleteServer(context.Background(), &server_managementv1.DeleteServerRequest{ServerId: "id-1"})

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestHandler_UpdateServer_Success(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	input := service.UpdateServerInput{ServerName: "new-srv", IPv4: "4.5.6.7", HealthCheckMethod: "ICMP"}
	svc.On("UpdateServer", mock.Anything, "uuid-1", input).
		Return(&domain.Server{ServerID: "uuid-1", ServerName: "new-srv", IPv4: "4.5.6.7"}, nil).Once()

	resp, err := h.UpdateServer(context.Background(), &server_managementv1.UpdateServerRequest{
		ServerId: "uuid-1", ServerName: "new-srv", Ipv4: "4.5.6.7",
	})

	assert.NoError(t, err)
	assert.Equal(t, "new-srv", resp.Server.ServerName)
}

func TestHandler_UpdateServer_NotFound(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("UpdateServer", mock.Anything, "uuid-missing", mock.Anything).
		Return(nil, service.ErrServerNotFound).Once()

	_, err := h.UpdateServer(context.Background(), &server_managementv1.UpdateServerRequest{
		ServerId: "uuid-missing", ServerName: "x", Ipv4: "1.1.1.1",
	})

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestHandler_DeleteServer_Success(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("DeleteServer", mock.Anything, "uuid-1").Return(nil).Once()

	resp, err := h.DeleteServer(context.Background(), &server_managementv1.DeleteServerRequest{ServerId: "uuid-1"})

	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestHandler_ViewServers_Success(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	servers := []*domain.Server{{ServerID: "id-1", ServerName: "srv", IPv4: "1.1.1.1"}}
	filter := repository.ServerListFilter{Page: 1, PageSize: 20}
	svc.On("SearchServers", mock.Anything, filter).Return(servers, int32(1), nil).Once()

	resp, err := h.ViewServers(context.Background(), &server_managementv1.ViewServersRequest{Page: 1, Limit: 20})

	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.TotalCount)
	assert.Len(t, resp.Servers, 1)
}

func TestHandler_ViewServers_Error(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("SearchServers", mock.Anything, mock.Anything).Return(nil, int32(0), errors.New("db error")).Once()

	_, err := h.ViewServers(context.Background(), &server_managementv1.ViewServersRequest{Page: 1, Limit: 20})

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestMapServerToPB_Nil(t *testing.T) {
	assert.Nil(t, mapServerToPB(nil))
}

func TestMapServerToPB_Full(t *testing.T) {
	now := time.Now()
	s := &domain.Server{
		ServerID: "uuid", ServerName: "name", IPv4: "1.2.3.4",
		CurrentStatus: domain.ServerStatusOffline,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	pb := mapServerToPB(s)

	assert.Equal(t, "uuid", pb.ServerId)
	assert.Equal(t, "OFFLINE", pb.CurrentStatus)
	assert.Equal(t, "name", pb.ServerName)
	assert.Equal(t, "1.2.3.4", pb.Ipv4)
	assert.Equal(t, now.Format(time.RFC3339), pb.CreatedAt)
	assert.Equal(t, now.Format(time.RFC3339), pb.UpdatedAt)
}

func TestHandler_NilRequests(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)
	ctx := context.Background()

	_, err := h.CreateServer(ctx, nil)
	assert.Error(t, err)

	_, err = h.UpdateServer(ctx, nil)
	assert.Error(t, err)

	_, err = h.DeleteServer(ctx, nil)
	assert.Error(t, err)

	_, err = h.ViewServers(ctx, nil)
	assert.Error(t, err)

	_, err = h.ImportServers(ctx, nil)
	assert.Error(t, err)

	_, err = h.ExportServers(ctx, nil)
	assert.Error(t, err)
}

func TestHandler_ImportServers_Success(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("ImportServers", mock.Anything, []byte("data")).
		Return(&service.ImportResult{SuccessCount: 1}, nil).Once()

	resp, err := h.ImportServers(context.Background(), &server_managementv1.ImportServersRequest{
		FileContent: []byte("data"),
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.SuccessCount)
}

func TestHandler_ImportServers_Errors(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("ImportServers", mock.Anything, []byte("size")).
		Return(nil, service.ErrFileTooLarge).Once()

	_, err := h.ImportServers(context.Background(), &server_managementv1.ImportServersRequest{FileContent: []byte("size")})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	svc.On("ImportServers", mock.Anything, []byte("format")).
		Return(nil, service.ErrInvalidFormat).Once()

	_, err = h.ImportServers(context.Background(), &server_managementv1.ImportServersRequest{FileContent: []byte("format")})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	svc.On("ImportServers", mock.Anything, []byte("other")).
		Return(nil, errors.New("internal db error")).Once()

	_, err = h.ImportServers(context.Background(), &server_managementv1.ImportServersRequest{FileContent: []byte("other")})
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestHandler_ExportServers_Success(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("ExportServers", mock.Anything, mock.Anything).
		Return([]byte("excel_data"), "servers.xlsx", nil).Once()

	resp, err := h.ExportServers(context.Background(), &server_managementv1.ExportServersRequest{})

	assert.NoError(t, err)
	assert.Equal(t, []byte("excel_data"), resp.FileContent)
	assert.Equal(t, "servers.xlsx", resp.Filename)
}

func TestHandler_ExportServers_Error(t *testing.T) {
	svc := new(mockSvc)
	h := NewServerManagementServer(svc)

	svc.On("ExportServers", mock.Anything, mock.Anything).
		Return(nil, "", errors.New("export failed")).Once()

	_, err := h.ExportServers(context.Background(), &server_managementv1.ExportServersRequest{})
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestParseCreatedDateRange(t *testing.T) {
	t.Run("valid dates", func(t *testing.T) {
		from, to, err := parseCreatedDateRange("2026-01-01", "2026-01-31")
		assert.NoError(t, err)
		assert.False(t, from.IsZero())
		assert.False(t, to.IsZero())
		assert.True(t, from.Before(to))
	})

	t.Run("empty dates", func(t *testing.T) {
		from, to, err := parseCreatedDateRange("", "")
		assert.NoError(t, err)
		assert.True(t, from.IsZero())
		assert.True(t, to.IsZero())
	})

	t.Run("invalid from format", func(t *testing.T) {
		_, _, err := parseCreatedDateRange("01-01-2026", "")
		assert.Error(t, err)
	})

	t.Run("invalid to format", func(t *testing.T) {
		_, _, err := parseCreatedDateRange("", "2026/01/01")
		assert.Error(t, err)
	})

	t.Run("from after to", func(t *testing.T) {
		_, _, err := parseCreatedDateRange("2026-02-01", "2026-01-01")
		assert.Error(t, err)
	})
}
