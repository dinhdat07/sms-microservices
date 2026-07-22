package impl

import (
	"context"
	"regexp"
	"testing"
	"time"

	"sms-reporting/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/google/uuid"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	assert.NoError(t, err)

	return gormDB, mock
}

func TestReportingRepository_CreateReportRequest(t *testing.T) {
	db, mock := setupDB(t)
	repo := NewGormReportingRepository(db)
	ctx := context.Background()

	reqID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	req := &domain.ReportRequest{
		ID:             reqID,
		RequestorEmail: "admin@example.com",
		Status:         domain.ReportStatusProcessing,
		CorrelationID:  "corr-1",
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "reporting_schema"."report_requests" ("id","requestor_email","start_time","end_time","status","correlation_id","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`)).
		WithArgs(req.ID, req.RequestorEmail, sqlmock.AnyArg(), sqlmock.AnyArg(), req.Status, req.CorrelationID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateReportRequest(ctx, req)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReportingRepository_UpdateReportStatus(t *testing.T) {
	db, mock := setupDB(t)
	repo := NewGormReportingRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "reporting_schema"."report_requests" SET "status"=$1,"updated_at"=$2 WHERE id = $3`)).
		WithArgs(domain.ReportStatusCompleted, sqlmock.AnyArg(), "req-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateReportStatus(ctx, "req-1", domain.ReportStatusCompleted)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReportingRepository_GetServerCountByStatus(t *testing.T) {
	db, mock := setupDB(t)
	repo := NewGormReportingRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "reporting_schema"."reporting_servers" WHERE status = $1`)).
		WithArgs("Healthy").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	count, err := repo.GetServerCountByStatus(ctx, "Healthy")
	assert.NoError(t, err)
	assert.Equal(t, int64(10), count)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "reporting_schema"."reporting_servers"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))

	count, err = repo.GetServerCountByStatus(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(50), count)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReportingRepository_UpsertReportingServer(t *testing.T) {
	db, mock := setupDB(t)
	repo := NewGormReportingRepository(db)
	ctx := context.Background()

	server := &domain.ReportingServer{
		ServerID:  "srv-1",
		Name:      "Server 1",
		IPv4:      "10.0.0.1",
		Status:    "Healthy",
		UpdatedAt: time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "reporting_schema"."reporting_servers" ("server_id","name","ipv4","status","updated_at") VALUES ($1,$2,$3,$4,$5) ON CONFLICT ("server_id") DO UPDATE SET "name"="excluded"."name","ipv4"="excluded"."ipv4","status"="excluded"."status","updated_at"="excluded"."updated_at"`)).
		WithArgs(server.ServerID, server.Name, server.IPv4, server.Status, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpsertReportingServer(ctx, server)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReportingRepository_UpdateReportingServerStatus(t *testing.T) {
	db, mock := setupDB(t)
	repo := NewGormReportingRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "reporting_schema"."reporting_servers" SET "status"=$1,"updated_at"=$2 WHERE server_id = $3`)).
		WithArgs("Unhealthy", sqlmock.AnyArg(), "srv-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateReportingServerStatus(ctx, "srv-1", "Unhealthy")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReportingRepository_DeleteReportingServer(t *testing.T) {
	db, mock := setupDB(t)
	repo := NewGormReportingRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "reporting_schema"."reporting_servers" WHERE server_id = $1`)).
		WithArgs("srv-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteReportingServer(ctx, "srv-1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
