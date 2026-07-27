package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"sms-management/internal/domain"
	"sms-management/internal/infrastructure/security"
	"sms-management/internal/repository"
)

var (
	ErrServerNotFound = errors.New("server not found")
	ErrIPv4Exists     = errors.New("ipv4 already exists")
	ErrNameExists     = errors.New("server name already exists")
)

type CreateServerInput struct {
	ServerName        string
	IPv4              string
	HealthCheckMethod domain.HealthCheckMethod
	SSHPort           int
	SSHUser           string
	SSHKey            string
	AgentEndpoint     string
}

type UpdateServerInput struct {
	ServerName        string
	IPv4              string
	HealthCheckMethod domain.HealthCheckMethod
	SSHPort           int
	SSHUser           string
	SSHKey            string
	AgentEndpoint     string
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
	repo       repository.ServerRepository
	outboxRepo repository.OutboxRepository
}

func NewServerService(repo repository.ServerRepository, outboxRepo repository.OutboxRepository) ServerService {
	return &serverService{
		repo:       repo,
		outboxRepo: outboxRepo,
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

	if input.HealthCheckMethod == domain.MethodSSH {
		if input.SSHPort <= 0 || input.SSHUser == "" || input.SSHKey == "" {
			return nil, errors.New("ssh_port, ssh_user, and ssh_key are required for SSH health check")
		}
	}
	if input.HealthCheckMethod == domain.MethodAgentPull {
		if input.AgentEndpoint == "" {
			return nil, errors.New("agent_endpoint is required for AGENT_PULL health check")
		}
	}

	// Clean irrelevant fields
	if input.HealthCheckMethod != domain.MethodSSH {
		input.SSHPort = 0
		input.SSHUser = ""
		input.SSHKey = ""
	}
	if input.HealthCheckMethod != domain.MethodAgentPull {
		input.AgentEndpoint = ""
	}

	encryptedKey, err := security.Encrypt(input.SSHKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt ssh key: %w", err)
	}

	server := &domain.Server{
		ServerName:        input.ServerName,
		IPv4:              input.IPv4,
		CurrentStatus:     domain.ServerStatusUnknown,
		HealthCheckMethod: input.HealthCheckMethod,
		SSHPort:           input.SSHPort,
		SSHUser:           input.SSHUser,
		SSHKey:            encryptedKey,
		AgentEndpoint:     input.AgentEndpoint,
	}

	err = s.repo.ExecuteInTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, server); err != nil {
			return err
		}

		payload, _ := json.Marshal(server)
		event := domain.NewOutboxEvent("Server", server.ServerID, domain.EventServerCreated, payload)
		return s.outboxRepo.Create(txCtx, event)
	})

	if err != nil {
		return nil, err
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

	if input.SSHPort == 0 {
		input.SSHPort = server.SSHPort
	}
	if input.SSHUser == "" {
		input.SSHUser = server.SSHUser
	}
	if input.AgentEndpoint == "" {
		input.AgentEndpoint = server.AgentEndpoint
	}

	if input.HealthCheckMethod == domain.MethodSSH {
		if input.SSHPort <= 0 || input.SSHUser == "" {
			return nil, errors.New("ssh_port and ssh_user are required for SSH health check")
		}
	}
	if input.HealthCheckMethod == domain.MethodAgentPull {
		if input.AgentEndpoint == "" {
			return nil, errors.New("agent_endpoint is required for AGENT_PULL health check")
		}
	}

	var encryptedKey string
	if input.SSHKey != "" {
		encryptedKey, err = security.Encrypt(input.SSHKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt ssh key: %w", err)
		}
	} else {
		encryptedKey = server.SSHKey
	}

	statusReset := server.HealthCheckMethod != input.HealthCheckMethod

	server.ServerName = input.ServerName
	server.HealthCheckMethod = input.HealthCheckMethod
	server.SSHPort = input.SSHPort
	server.SSHUser = input.SSHUser
	server.SSHKey = encryptedKey
	server.AgentEndpoint = input.AgentEndpoint

	if server.IPv4 != input.IPv4 {
		existingIP, err := s.repo.GetByIPv4(ctx, input.IPv4)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		if existingIP != nil && existingIP.ServerID != id {
			return nil, ErrIPv4Exists
		}
		server.IPv4 = input.IPv4
		statusReset = true
	}

	if statusReset {
		server.CurrentStatus = domain.ServerStatusUnknown
	}

	err = s.repo.ExecuteInTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Update(txCtx, server); err != nil {
			return err
		}

		payload, _ := json.Marshal(server)
		event := domain.NewOutboxEvent("Server", server.ServerID, domain.EventServerUpdated, payload)
		return s.outboxRepo.Create(txCtx, event)
	})

	if err != nil {
		return nil, err
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

	err = s.repo.ExecuteInTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Delete(txCtx, id); err != nil {
			return err
		}

		payload := []byte(`{"server_id":"` + id + `"}`)
		event := domain.NewOutboxEvent("Server", id, domain.EventServerDeleted, payload)
		return s.outboxRepo.Create(txCtx, event)
	})

	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrServerNotFound
		}
		return err
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
