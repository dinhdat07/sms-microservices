package repository

import (
	"context"
	"time"

	"sms-management/internal/modules/server_management/domain"
)

type ServerListFilter struct {
	Page          int
	PageSize      int
	Status        string
	Name          string
	CreatedFrom   time.Time
	CreatedTo     time.Time
	SortBy        string
	SortDirection string // "asc" or "desc"
}

type ServerRepository interface {
	Create(ctx context.Context, server *domain.Server) error

	GetByID(ctx context.Context, id string) (*domain.Server, error)
	GetByIPv4(ctx context.Context, ipv4 string) (*domain.Server, error)
	GetByName(ctx context.Context, name string) (*domain.Server, error)
	FindByNamesOrIPv4s(ctx context.Context, names []string, ipv4s []string) ([]*domain.Server, error)

	Update(ctx context.Context, server *domain.Server) error
	Delete(ctx context.Context, id string) error

	BatchCreate(ctx context.Context, servers []*domain.Server) error
	Search(ctx context.Context, filter ServerListFilter) ([]*domain.Server, int32, error)
}
