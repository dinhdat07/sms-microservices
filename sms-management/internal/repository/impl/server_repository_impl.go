package impl

import (
	"context"
	"errors"
	"strings"

	"sms-management/internal/domain"
	"sms-management/internal/repository"

	"gorm.io/gorm"
)

type GormServerRepository struct {
	db *gorm.DB
}

func NewGormServerRepository(db *gorm.DB) repository.ServerRepository {
	return &GormServerRepository{db: db}
}

func (r *GormServerRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GormServerRepository) Create(ctx context.Context, server *domain.Server) error {
	server.ServerName = strings.TrimSpace(server.ServerName)
	server.IPv4 = strings.TrimSpace(server.IPv4)
	return r.getDB(ctx).Create(server).Error
}

func (r *GormServerRepository) GetByID(ctx context.Context, id string) (*domain.Server, error) {
	var server domain.Server
	err := r.getDB(ctx).First(&server, "server_id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &server, nil
}

func (r *GormServerRepository) GetByIPv4(ctx context.Context, ipv4 string) (*domain.Server, error) {
	var server domain.Server
	err := r.getDB(ctx).Where("ipv4 = ?", strings.TrimSpace(ipv4)).First(&server).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &server, nil
}

func (r *GormServerRepository) GetByName(ctx context.Context, name string) (*domain.Server, error) {
	var server domain.Server
	err := r.getDB(ctx).Where("server_name = ?", strings.TrimSpace(name)).First(&server).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &server, nil
}

func (r *GormServerRepository) FindByNamesOrIPv4s(ctx context.Context, names []string, ipv4s []string) ([]*domain.Server, error) {
	if len(names) == 0 && len(ipv4s) == 0 {
		return nil, nil
	}
	var servers []*domain.Server
	err := r.getDB(ctx).Where("server_name IN ? OR ipv4 IN ?", names, ipv4s).Find(&servers).Error
	if err != nil {
		return nil, err
	}
	return servers, nil
}

func (r *GormServerRepository) Update(ctx context.Context, server *domain.Server) error {
	server.ServerName = strings.TrimSpace(server.ServerName)
	server.IPv4 = strings.TrimSpace(server.IPv4)

	result := r.getDB(ctx).Model(server).Select("server_name", "ipv4", "current_status", "consecutive_failures").Updates(server)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *GormServerRepository) Delete(ctx context.Context, id string) error {
	result := r.getDB(ctx).Unscoped().Delete(&domain.Server{}, "server_id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *GormServerRepository) BatchCreate(ctx context.Context, servers []*domain.Server) error {
	if len(servers) == 0 {
		return nil
	}
	return r.getDB(ctx).Create(&servers).Error
}

func (r *GormServerRepository) Search(ctx context.Context, filter repository.ServerListFilter) ([]*domain.Server, int32, error) {
	query := r.getDB(ctx).Model(&domain.Server{})

	if filter.Status != "" {
		query = query.Where("current_status = ?", filter.Status)
	}
	if filter.Name != "" {
		term := "%" + strings.TrimSpace(filter.Name) + "%"
		query = query.Where("server_name ILIKE ? OR ipv4 ILIKE ?", term, term)
	}
	if !filter.CreatedFrom.IsZero() {
		query = query.Where("created_at >= ?", filter.CreatedFrom)
	}
	if !filter.CreatedTo.IsZero() {
		query = query.Where("created_at < ?", filter.CreatedTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.SortBy != "" {
		direction := "asc"
		if strings.ToLower(filter.SortDirection) == "desc" {
			direction = "desc"
		}
		query = query.Order(filter.SortBy + " " + direction)
	} else {
		query = query.Order("created_at desc")
	}

	offset := (filter.Page - 1) * filter.PageSize
	query = query.Offset(offset).Limit(filter.PageSize)

	var servers []*domain.Server
	if err := query.Find(&servers).Error; err != nil {
		return nil, 0, err
	}

	return servers, int32(total), nil
}
