package impl

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"sms-identity/internal/repository"
)

func setupUserRepoMock(t *testing.T) (repository.UserRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	return NewUserRepository(gormDB), mock
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	repo, mock := setupUserRepoMock(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnError(gorm.ErrRecordNotFound)

	user, err := repo.FindByEmail(context.Background(), "test@test.com")
	assert.NoError(t, err) // returns nil, nil
	assert.Nil(t, user)
}

func TestUserRepository_FindByEmail_Error(t *testing.T) {
	repo, mock := setupUserRepoMock(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnError(errors.New("db error"))

	user, err := repo.FindByEmail(context.Background(), "test@test.com")
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	repo, mock := setupUserRepoMock(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnError(gorm.ErrRecordNotFound)

	user, err := repo.FindByID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestUserRepository_FindByID_Error(t *testing.T) {
	repo, mock := setupUserRepoMock(t)

	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnError(errors.New("db error"))

	user, err := repo.FindByID(context.Background(), 1)
	assert.Error(t, err)
	assert.Nil(t, user)
}