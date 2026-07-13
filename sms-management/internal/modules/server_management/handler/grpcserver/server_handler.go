package grpcserver

import (
	"context"
	"errors"
	"time"

	server_managementv1 "sms-management/gen/go/server_management/v1"
	"sms-management/internal/modules/server_management/domain"
	"sms-management/internal/modules/server_management/repository"
	"sms-management/internal/modules/server_management/service"

	"google.golang.org/grpc/codes"
	gstatus "google.golang.org/grpc/status"
)

type ServerManagementServer struct {
	server_managementv1.UnimplementedServerManagementServiceServer
	serverService service.ServerService
}

func NewServerManagementServer(serverService service.ServerService) *ServerManagementServer {
	return &ServerManagementServer{
		serverService: serverService,
	}
}

func mapError(err error) error {
	if errors.Is(err, service.ErrServerNotFound) {
		return gstatus.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, service.ErrIPv4Exists) || errors.Is(err, service.ErrNameExists) {
		return gstatus.Error(codes.AlreadyExists, err.Error())
	}
	if errors.Is(err, service.ErrFileTooLarge) ||
		errors.Is(err, service.ErrInvalidFormat) ||
		errors.Is(err, service.ErrNoSheets) ||
		errors.Is(err, service.ErrEmptyFile) ||
		errors.Is(err, service.ErrMissingCols) {
		return gstatus.Error(codes.InvalidArgument, err.Error())
	}
	return gstatus.Error(codes.Internal, err.Error())
}

func mapServerToPB(server *domain.Server) *server_managementv1.Server {
	if server == nil {
		return nil
	}
	return &server_managementv1.Server{
		ServerId:      server.ServerID,
		ServerName:    server.ServerName,
		Ipv4:          server.IPv4,
		CurrentStatus: string(server.CurrentStatus),
		CreatedAt:     server.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     server.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *ServerManagementServer) CreateServer(ctx context.Context, req *server_managementv1.CreateServerRequest) (*server_managementv1.CreateServerResponse, error) {
	if req == nil {
		return nil, gstatus.Error(codes.InvalidArgument, "request is required")
	}

	input := service.CreateServerInput{
		ServerName: req.GetServerName(),
		IPv4:       req.GetIpv4(),
	}

	server, err := s.serverService.CreateServer(ctx, input)
	if err != nil {
		return nil, mapError(err)
	}

	return &server_managementv1.CreateServerResponse{
		Server: mapServerToPB(server),
	}, nil
}

func (s *ServerManagementServer) UpdateServer(ctx context.Context, req *server_managementv1.UpdateServerRequest) (*server_managementv1.UpdateServerResponse, error) {
	if req == nil {
		return nil, gstatus.Error(codes.InvalidArgument, "request is required")
	}

	input := service.UpdateServerInput{
		ServerName: req.GetServerName(),
		IPv4:       req.GetIpv4(),
	}

	server, err := s.serverService.UpdateServer(ctx, req.GetServerId(), input)
	if err != nil {
		return nil, mapError(err)
	}

	return &server_managementv1.UpdateServerResponse{
		Server: mapServerToPB(server),
	}, nil
}

func (s *ServerManagementServer) DeleteServer(ctx context.Context, req *server_managementv1.DeleteServerRequest) (*server_managementv1.DeleteServerResponse, error) {
	if req == nil {
		return nil, gstatus.Error(codes.InvalidArgument, "request is required")
	}

	err := s.serverService.DeleteServer(ctx, req.GetServerId())
	if err != nil {
		return nil, mapError(err)
	}

	return &server_managementv1.DeleteServerResponse{
		Success: true,
	}, nil
}

func (s *ServerManagementServer) ViewServers(ctx context.Context, req *server_managementv1.ViewServersRequest) (*server_managementv1.ViewServersResponse, error) {
	if req == nil {
		return nil, gstatus.Error(codes.InvalidArgument, "request is required")
	}

	createdFrom, createdTo, err := parseCreatedDateRange(req.GetCreatedFrom(), req.GetCreatedTo())
	if err != nil {
		return nil, err
	}

	filter := repository.ServerListFilter{
		Page:          int(req.GetPage()),
		PageSize:      int(req.GetLimit()),
		Status:        req.GetFilterStatus(),
		Name:          req.GetFilterName(),
		CreatedFrom:   createdFrom,
		CreatedTo:     createdTo,
		SortBy:        req.GetSortBy(),
		SortDirection: req.GetSortDirection(),
	}

	servers, totalCount, err := s.serverService.SearchServers(ctx, filter)
	if err != nil {
		return nil, mapError(err)
	}

	var pbServers []*server_managementv1.Server
	for _, server := range servers {
		pbServers = append(pbServers, mapServerToPB(server))
	}

	return &server_managementv1.ViewServersResponse{
		TotalCount: totalCount,
		Servers:    pbServers,
	}, nil
}

func (s *ServerManagementServer) ImportServers(ctx context.Context, req *server_managementv1.ImportServersRequest) (*server_managementv1.ImportServersResponse, error) {
	if req == nil {
		return nil, gstatus.Error(codes.InvalidArgument, "request is required")
	}

	result, err := s.serverService.ImportServers(ctx, req.GetFileContent())
	if err != nil {
		return nil, mapError(err)
	}

	return &server_managementv1.ImportServersResponse{
		SuccessCount:      result.SuccessCount,
		SuccessfulServers: result.SuccessfulServers,
		FailCount:         result.FailCount,
		FailedServers:     result.FailedServers,
	}, nil
}

func (s *ServerManagementServer) ExportServers(ctx context.Context, req *server_managementv1.ExportServersRequest) (*server_managementv1.ExportServersResponse, error) {
	if req == nil {
		return nil, gstatus.Error(codes.InvalidArgument, "request is required")
	}

	createdFrom, createdTo, err := parseCreatedDateRange(req.GetCreatedFrom(), req.GetCreatedTo())
	if err != nil {
		return nil, err
	}

	filter := repository.ServerListFilter{
		Page:          int(req.GetPage()),
		PageSize:      int(req.GetLimit()),
		Status:        req.GetFilterStatus(),
		Name:          req.GetFilterName(),
		CreatedFrom:   createdFrom,
		CreatedTo:     createdTo,
		SortBy:        req.GetSortBy(),
		SortDirection: req.GetSortDirection(),
	}

	fileBytes, filename, err := s.serverService.ExportServers(ctx, filter)
	if err != nil {
		return nil, mapError(err)
	}

	return &server_managementv1.ExportServersResponse{
		FileContent: fileBytes,
		Filename:    filename,
	}, nil
}

func parseCreatedDateRange(createdFrom, createdTo string) (time.Time, time.Time, error) {
	var from time.Time
	var to time.Time
	var err error
	if createdFrom != "" {
		from, err = time.Parse("2006-01-02", createdFrom)
		if err != nil {
			return time.Time{}, time.Time{}, gstatus.Error(codes.InvalidArgument, "created_from must use YYYY-MM-DD format")
		}
	}
	if createdTo != "" {
		to, err = time.Parse("2006-01-02", createdTo)
		if err != nil {
			return time.Time{}, time.Time{}, gstatus.Error(codes.InvalidArgument, "created_to must use YYYY-MM-DD format")
		}
		to = to.AddDate(0, 0, 1)
	}
	if !from.IsZero() && !to.IsZero() && !from.Before(to) {
		return time.Time{}, time.Time{}, gstatus.Error(codes.InvalidArgument, "created_from must be before or equal to created_to")
	}
	return from, to, nil
}
