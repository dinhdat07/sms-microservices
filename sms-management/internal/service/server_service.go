package service

import (
	"context"
	"errors"
	"sms-management/internal/infrastructure/logger"

	"sms-management/internal/infrastructure/redis"
	"sms-management/internal/domain"
	"sms-management/internal/repository"
)

var (
	ErrServerNotFound = errors.New("server not found")
	ErrIPv4Exists     = errors.New("ipv4 already exists")
	ErrNameExists     = errors.New("server name already exists")
)

type CreateServerInput struct {
	ServerName string
	IPv4       string
}

type UpdateServerInput struct {
	ServerName string
	IPv4       string
}

type ImportResult struct {
	SuccessCount      int32
	SuccessfulServers []string
	FailCount         int32
	FailedServers     []string
}

type ServerService interface {
	CreateServer(ctx context.Context, input CreateServerInput) (*domain.Server, error)
	UpdateServer(ctx context.Context, id string, input UpdateServerInput) (*domain.Server, error)
	DeleteServer(ctx context.Context, id string) error
	SearchServers(ctx context.Context, filter repository.ServerListFilter) ([]*domain.Server, int32, error)

	ImportServers(ctx context.Context, fileBytes []byte) (*ImportResult, error)
	ExportServers(ctx context.Context, filter repository.ServerListFilter) ([]byte, string, error)
}

type serverService struct {
	repo  repository.ServerRepository
	cache redis.CacheManager
}

func NewServerService(repo repository.ServerRepository, cache redis.CacheManager) ServerService {
	return &serverService{
		repo:  repo,
		cache: cache,
	}
}

func (s *serverService) CreateServer(ctx context.Context, input CreateServerInput) (*domain.Server, error) {
	existingName, err := s.repo.GetByName(ctx, input.ServerName)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if existingName != nil {
		return nil, ErrNameExists
	}

	existingIP, err := s.repo.GetByIPv4(ctx, input.IPv4)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if existingIP != nil {
		return nil, ErrIPv4Exists
	}

	server := &domain.Server{
		ServerName: input.ServerName,
		IPv4:       input.IPv4,
	}

	err = s.repo.Create(ctx, server)
	if err != nil {
		return nil, err
	}

	// Dual-Write to Redis
	if s.cache != nil {
		if err := s.cache.Upsert(ctx, server.ServerID, server.IPv4, string(server.CurrentStatus), 0); err != nil {
			logger.Log.Sugar().Errorf("[WARNING] DB Create succeeded but Redis sync failed for ServerID %s: %v", server.ServerID, err)
		}
	}

	return server, nil
}

func (s *serverService) UpdateServer(ctx context.Context, id string, input UpdateServerInput) (*domain.Server, error) {
	server, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrServerNotFound
		}
		return nil, err
	}
	if server == nil {
		return nil, ErrServerNotFound
	}

	if input.ServerName != server.ServerName {
		existingName, err := s.repo.GetByName(ctx, input.ServerName)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		if existingName != nil && existingName.ServerID != id {
			return nil, ErrNameExists
		}
		server.ServerName = input.ServerName
	}

	if input.IPv4 != server.IPv4 {
		existingIP, err := s.repo.GetByIPv4(ctx, input.IPv4)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		if existingIP != nil && existingIP.ServerID != id {
			return nil, ErrIPv4Exists
		}
		server.IPv4 = input.IPv4
	}

	err = s.repo.Update(ctx, server)
	if err != nil {
		return nil, err
	}

	// Dual-Write to Redis
	if s.cache != nil {
		if err := s.cache.Upsert(ctx, server.ServerID, server.IPv4, string(server.CurrentStatus), server.ConsecutiveFailures); err != nil {
			logger.Log.Sugar().Errorf("[WARNING] DB Update succeeded but Redis sync failed for ServerID %s: %v", server.ServerID, err)
		}
	}

	return server, nil
}

func (s *serverService) DeleteServer(ctx context.Context, id string) error {
	server, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrServerNotFound
		}
		return err
	}
	if server == nil {
		return ErrServerNotFound
	}

	err = s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	// Dual-Write to Redis
	if s.cache != nil {
		if err := s.cache.Delete(ctx, id); err != nil {
			logger.Log.Sugar().Errorf("[WARNING] DB Delete succeeded but Redis sync failed for ServerID %s: %v", id, err)
		}
	}

	return nil
}

func (s *serverService) SearchServers(ctx context.Context, filter repository.ServerListFilter) ([]*domain.Server, int32, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	if filter.Status != "" && !domain.ServerStatus(filter.Status).IsValid() {
		return nil, 0, errors.New("invalid status filter")
	}

	return s.repo.Search(ctx, filter)
}
