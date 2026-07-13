package impl

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGormTxManager_WithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	assert.NoError(t, err)

	txManager := NewGormTxManager(gdb)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit()

		err := txManager.WithTx(context.Background(), func(ctx context.Context) error {
			txValue := ctx.Value(txKey{})
			assert.NotNil(t, txValue)
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("Rollback on error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()

		err := txManager.WithTx(context.Background(), func(ctx context.Context) error {
			return errors.New("some error")
		})
		assert.Error(t, err)
	})
}
