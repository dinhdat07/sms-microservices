package impl

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	
	"sms-identity/internal/domain"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	dialector := postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	})

	gormDB, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	return gormDB, mock
}

func TestGormIdentityRepository_UserRepository(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("FindByEmail", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM .*users.* WHERE email = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow(1, "test@test.com"))

		user, err := repo.FindByEmail(context.Background(), "test@test.com")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, uint(1), user.ID)
	})

	t.Run("FindByID", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM .*users.* WHERE id = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow(1, "test@test.com"))

		user, err := repo.FindByID(context.Background(), 1)
		assert.NoError(t, err)
		assert.NotNil(t, user)
	})
}

func TestGormIdentityRepository_AuthSessionRepository(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewAuthSessionRepository(db)

	id := uuid.New()
	session := &domain.AuthSession{
		ID:        id,
		UserID:    1,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	t.Run("Create", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO .*auth_sessions.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.String()))

		err := repo.Create(context.Background(), session)
		assert.NoError(t, err)
	})

	t.Run("FindActiveByID", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM .*auth_sessions.* WHERE id = \$1 AND revoked_at IS NULL AND expires_at > \$2.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.String()))

		res, err := repo.FindActiveByID(context.Background(), id)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("ListActiveByUserID", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM .*auth_sessions.* WHERE user_id = \$1 AND revoked_at IS NULL AND expires_at > \$2`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.String()))

		res, err := repo.ListActiveByUserID(context.Background(), 1)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("RevokeByID", func(t *testing.T) {
		mock.ExpectExec(`UPDATE .*auth_sessions.*`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.RevokeByID(context.Background(), id)
		assert.NoError(t, err)
	})

	t.Run("RevokeAllByUserID", func(t *testing.T) {
		mock.ExpectExec(`UPDATE .*auth_sessions.*`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.RevokeAllByUserID(context.Background(), 1)
		assert.NoError(t, err)
	})
}

func TestGormIdentityRepository_RefreshTokenRepository(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)

	token := &domain.RefreshToken{
		ID:        uuid.New(),
		SessionID: uuid.New(),
		TokenHash: "hash123",
	}

	t.Run("Create", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO .*refresh_tokens.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(token.ID.String()))

		err := repo.Create(context.Background(), token)
		assert.NoError(t, err)
	})

	t.Run("FindByTokenHash", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM .*refresh_tokens.* WHERE token_hash = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(token.ID.String()))

		res, err := repo.FindByTokenHash(context.Background(), "hash123")
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("RevokeByID", func(t *testing.T) {
		mock.ExpectExec(`UPDATE .*refresh_tokens.*`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.RevokeByID(context.Background(), token.ID)
		assert.NoError(t, err)
	})

	t.Run("RevokeBySessionID", func(t *testing.T) {
		mock.ExpectExec(`UPDATE .*refresh_tokens.*`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.RevokeBySessionID(context.Background(), token.SessionID)
		assert.NoError(t, err)
	})

	t.Run("RevokeByUserID", func(t *testing.T) {
		mock.ExpectExec(`UPDATE .*refresh_tokens.*`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.RevokeByUserID(context.Background(), 1)
		assert.NoError(t, err)
	})

	t.Run("MarkReplacement", func(t *testing.T) {
		mock.ExpectExec(`UPDATE .*refresh_tokens.*`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.MarkReplacement(context.Background(), token.ID, uuid.New())
		assert.NoError(t, err)
	})
}