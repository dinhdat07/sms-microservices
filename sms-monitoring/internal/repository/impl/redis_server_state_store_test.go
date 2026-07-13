package impl

import (
	"context"
	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"

	"sms-monitoring/internal/repository"
)

func TestRedisServerStateStore_GetServerState_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewRedisServerStateStore(db)
	ctx := context.Background()

	mock.ExpectHGetAll("server:info:srv-1").SetVal(map[string]string{
		"status":      "ONLINE",
		"retry_count": "2",
	})

	state, err := store.GetServerState(ctx, "srv-1")
	assert.NoError(t, err)
	assert.Equal(t, "ONLINE", state.Status)
	assert.Equal(t, 2, state.RetryCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisServerStateStore_GetServerState_NotFound(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewRedisServerStateStore(db)
	ctx := context.Background()

	mock.ExpectHGetAll("server:info:srv-2").SetVal(map[string]string{})

	state, err := store.GetServerState(ctx, "srv-2")
	assert.ErrorIs(t, err, repository.ErrServerStateNotFound)
	assert.Nil(t, state)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisServerStateStore_GetServerState_Error(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewRedisServerStateStore(db)
	ctx := context.Background()

	mock.ExpectHGetAll("server:info:srv-3").SetErr(errors.New("redis error"))

	state, err := store.GetServerState(ctx, "srv-3")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis error")
	assert.Nil(t, state)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisServerStateStore_SetServerState_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewRedisServerStateStore(db)
	ctx := context.Background()

	mock.ExpectHSet("server:info:srv-4", "status", "OFFLINE", "retry_count", 3).SetVal(2)

	err := store.SetServerState(ctx, "srv-4", "OFFLINE", 3)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRedisServerStateStore_SetServerState_Error(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewRedisServerStateStore(db)
	ctx := context.Background()

	mock.ExpectHSet("server:info:srv-5", "status", "ONLINE", "retry_count", 0).SetErr(errors.New("write error"))

	err := store.SetServerState(ctx, "srv-5", "ONLINE", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write error")
	assert.NoError(t, mock.ExpectationsWereMet())
}
