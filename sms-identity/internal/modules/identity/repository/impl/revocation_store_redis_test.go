package impl

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSessionRevocationStore_MarkRevoked(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := NewSessionRevocationStore(db)

	sessionID := uuid.New()
	key := "revoked_session:" + sessionID.String()

	t.Run("success", func(t *testing.T) {
		expiresAt := time.Now().Add(1 * time.Hour)
		// Pass 1 hour to match the ttl passed to Set
		mock.ExpectSet(key, "revoked", 1*time.Hour).SetVal("OK")

		err := store.MarkRevoked(context.Background(), sessionID, expiresAt)
		assert.NoError(t, err)
	})

	t.Run("already expired", func(t *testing.T) {
		expiresAt := time.Now().Add(-1 * time.Hour)
		
		err := store.MarkRevoked(context.Background(), sessionID, expiresAt)
		assert.NoError(t, err)
	})
}