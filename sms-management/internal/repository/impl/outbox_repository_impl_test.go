package impl

import (
	"context"
	"regexp"
	"testing"
	"time"

	"sms-management/internal/domain"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupOutboxMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	return gormDB, mock
}

func TestOutboxRepository_Create(t *testing.T) {
	db, mock := setupOutboxMockDB(t)
	repo := NewGormOutboxRepository(db)

	evt := domain.NewOutboxEvent("Server", "srv-1", "ServerCreated", []byte(`{}`))
	
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outbox_events"`)).
		WithArgs(evt.ID, evt.AggregateType, evt.AggregateID, evt.EventType, evt.Payload, evt.IsProcessed, evt.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), evt)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_BatchCreate(t *testing.T) {
	db, mock := setupOutboxMockDB(t)
	repo := NewGormOutboxRepository(db)

	events := []*domain.OutboxEvent{
		domain.NewOutboxEvent("Server", "srv-1", "ServerCreated", []byte(`{}`)),
		domain.NewOutboxEvent("Server", "srv-2", "ServerCreated", []byte(`{}`)),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outbox_events"`)).
		WithArgs(
			events[0].ID, events[0].AggregateType, events[0].AggregateID, events[0].EventType, events[0].Payload, events[0].IsProcessed, events[0].CreatedAt,
			events[1].ID, events[1].AggregateType, events[1].AggregateID, events[1].EventType, events[1].Payload, events[1].IsProcessed, events[1].CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(2, 2))

	err := repo.BatchCreate(context.Background(), events)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_GetUnprocessed(t *testing.T) {
	db, mock := setupOutboxMockDB(t)
	repo := NewGormOutboxRepository(db)

	rows := sqlmock.NewRows([]string{"id", "aggregate_type", "aggregate_id", "event_type", "payload", "created_at"}).
		AddRow("evt-1", "Server", "srv-1", "ServerCreated", []byte(`{}`), time.Now()).
		AddRow("evt-2", "Server", "srv-2", "ServerCreated", []byte(`{}`), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "outbox_events" WHERE is_processed = $1 ORDER BY created_at ASC LIMIT $2`)).
		WithArgs(false, 10).
		WillReturnRows(rows)

	events, err := repo.GetUnprocessed(context.Background(), 10)
	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_MarkProcessed(t *testing.T) {
	db, mock := setupOutboxMockDB(t)
	repo := NewGormOutboxRepository(db)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "outbox_events" SET "is_processed"=$1 WHERE id IN ($2,$3)`)).
		WithArgs(sqlmock.AnyArg(), "evt-1", "evt-2").
		WillReturnResult(sqlmock.NewResult(2, 2))

	err := repo.MarkProcessed(context.Background(), []string{"evt-1", "evt-2"})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_Tx(t *testing.T) {
	db, mock := setupOutboxMockDB(t)
	repo := NewGormOutboxRepository(db)

	mock.ExpectBegin()
	tx := db.Begin()
	ctx := context.WithValue(context.Background(), txKey{}, tx)

	evt := domain.NewOutboxEvent("Server", "srv-1", "ServerCreated", []byte(`{}`))
	
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "outbox_events"`)).
		WithArgs(evt.ID, evt.AggregateType, evt.AggregateID, evt.EventType, evt.Payload, evt.IsProcessed, evt.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, evt)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
