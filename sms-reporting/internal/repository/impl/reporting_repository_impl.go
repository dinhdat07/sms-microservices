package impl

import (
	"context"
	"time"

	"sms-reporting/internal/domain"
	"sms-reporting/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormReportingRepository struct {
	db *gorm.DB
}

func NewGormReportingRepository(db *gorm.DB) repository.ReportingRepository {
	return &gormReportingRepository{
		db: db,
	}
}

func (r *gormReportingRepository) CreateReportRequest(ctx context.Context, req *domain.ReportRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *gormReportingRepository) UpdateReportStatus(ctx context.Context, reqID string, status string) error {
	return r.db.WithContext(ctx).Model(&domain.ReportRequest{}).Where("id = ?", reqID).Update("status", status).Error
}

// GetServerCountByStatus queries the LOCAL reporting_servers table (not management_schema).
func (r *gormReportingRepository) GetServerCountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.ReportingServer{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *gormReportingRepository) UpsertReportingServer(ctx context.Context, server *domain.ReportingServer) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "server_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "ipv4", "status", "updated_at"}),
	}).Create(server).Error
}

func (r *gormReportingRepository) UpdateReportingServerStatus(ctx context.Context, serverID string, status string) error {
	result := r.db.WithContext(ctx).Model(&domain.ReportingServer{}).Where("server_id = ?", serverID).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrRecordNotFound
	}
	return nil
}

func (r *gormReportingRepository) DeleteReportingServer(ctx context.Context, serverID string) error {
	return r.db.WithContext(ctx).Where("server_id = ?", serverID).Delete(&domain.ReportingServer{}).Error
}

func (r *gormReportingRepository) SaveDailyUptimeStat(ctx context.Context, stat *domain.DailyUptimeStat) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_ping_count", "success_ping_count", "updated_at"}),
	}).Create(stat).Error
}

func (r *gormReportingRepository) GetDailyUptimeStats(ctx context.Context, startDate time.Time, endDate time.Time) ([]domain.DailyUptimeStat, error) {
	var stats []domain.DailyUptimeStat
	err := r.db.WithContext(ctx).
		Where("date >= ? AND date <= ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Find(&stats).Error
	return stats, err
}
