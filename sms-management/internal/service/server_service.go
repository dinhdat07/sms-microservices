package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"sms-management/internal/domain"
	"sms-management/internal/infrastructure/security"
	"sms-management/internal/repository"
)

var (
	ErrServerNotFound = errors.New("Server not found.")
	ErrIPv4Exists     = errors.New("This IPv4 address already exists.")
	ErrNameExists     = errors.New("This Server name already exists.")

	ErrInvalidSSHPort             = errors.New("SSH port must be between 1 and 65535.")
	ErrInvalidSSHUser             = errors.New("SSH user is required for SSH health check.")
	ErrInvalidSSHKey              = errors.New("SSH private key is required for SSH health check.")
	ErrInvalidSSHKeyFormat        = errors.New("SSH private key must be a valid format (contains PRIVATE KEY).")
	ErrInvalidAgentEndpoint       = errors.New("Agent endpoint is required for AGENT_PULL.")
	ErrInvalidAgentEndpointFormat = errors.New("Agent endpoint must be a valid absolute URL.")
	ErrInvalidStatusFilter        = errors.New("Invalid status filter.")
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
		if input.SSHPort <= 0 || input.SSHPort > 65535 {
			return nil, ErrInvalidSSHPort
		}
		if input.SSHUser == "" {
			return nil, ErrInvalidSSHUser
		}
		if input.SSHKey == "" {
			return nil, ErrInvalidSSHKey
		}
		if !strings.Contains(input.SSHKey, "PRIVATE KEY") {
			return nil, ErrInvalidSSHKeyFormat
		}
	}
	if input.HealthCheckMethod == domain.MethodAgentPull {
		if input.AgentEndpoint == "" {
			return nil, ErrInvalidAgentEndpoint
		}
		if _, err := url.ParseRequestURI(input.AgentEndpoint); err != nil {
			return nil, ErrInvalidAgentEndpointFormat
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
		if input.SSHPort <= 0 || input.SSHPort > 65535 {
			return nil, ErrInvalidSSHPort
		}
		if input.SSHUser == "" {
			return nil, ErrInvalidSSHUser
		}
		if input.SSHKey != "" && !strings.Contains(input.SSHKey, "PRIVATE KEY") {
			return nil, ErrInvalidSSHKeyFormat
		}
	}
	if input.HealthCheckMethod == domain.MethodAgentPull {
		if input.AgentEndpoint == "" {
			return nil, ErrInvalidAgentEndpoint
		}
		if _, err := url.ParseRequestURI(input.AgentEndpoint); err != nil {
			return nil, ErrInvalidAgentEndpointFormat
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
		return nil, 0, ErrInvalidStatusFilter
	}

	return s.repo.Search(ctx, filter)
}
