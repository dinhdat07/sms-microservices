package impl

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"sms-management/internal/modules/server_management/domain"
	"sms-management/internal/modules/server_management/repository"
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

func TestGormServerRepository_GetByID(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewGormServerRepository(db)

	id := "server-1"

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"server_id", "server_name", "ipv4"}).
			AddRow(id, "Test Server", "192.168.1.1")

		mock.ExpectQuery(`SELECT \* FROM .*servers.* WHERE server_id = \$1.*`).
			WillReturnRows(rows)

		server, err := repo.GetByID(context.Background(), id)
		assert.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, id, server.ServerID)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM .*servers.* WHERE server_id = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"server_id", "server_name", "ipv4"}))

		server, err := repo.GetByID(context.Background(), id)
		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Nil(t, server)
	})
}

func TestGormServerRepository_GetByIPv4(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewGormServerRepository(db)

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"server_id", "server_name", "ipv4"}).
			AddRow("srv-1", "Test Server", "10.0.0.1")
		mock.ExpectQuery(`SELECT \* FROM .*servers.* WHERE ipv4 = \$1.*`).
			WillReturnRows(rows)

		server, err := repo.GetByIPv4(context.Background(), "10.0.0.1")
		assert.NoError(t, err)
		assert.NotNil(t, server)
	})
	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM .*servers.* WHERE ipv4 = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"server_id", "server_name", "ipv4"}))

		server, err := repo.GetByIPv4(context.Background(), "10.0.0.1")
		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Nil(t, server)
	})
}

func TestGormServerRepository_GetByName(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewGormServerRepository(db)

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"server_id", "server_name", "ipv4"}).
			AddRow("srv-1", "Test Server", "10.0.0.1")
		mock.ExpectQuery(`SELECT \* FROM .*servers.* WHERE server_name = \$1.*`).
			WillReturnRows(rows)

		server, err := repo.GetByName(context.Background(), "Test Server")
		assert.NoError(t, err)
		assert.NotNil(t, server)
	})
	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM .*servers.* WHERE server_name = \$1.*`).
			WillReturnRows(sqlmock.NewRows([]string{"server_id", "server_name", "ipv4"}))

		server, err := repo.GetByName(context.Background(), "Test Server")
		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Nil(t, server)
	})
}

func TestGormServerRepository_FindByNamesOrIPv4s(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewGormServerRepository(db)

	t.Run("empty", func(t *testing.T) {
		servers, err := repo.FindByNamesOrIPv4s(context.Background(), nil, nil)
		assert.NoError(t, err)
		assert.Nil(t, servers)
	})

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"server_id", "server_name", "ipv4"}).
			AddRow("srv-1", "Test Server", "10.0.0.1")
		mock.ExpectQuery(`SELECT \* FROM .*servers.* WHERE server_name IN \(\$1\) OR ipv4 IN \(\$2\).*`).
			WillReturnRows(rows)

		servers, err := repo.FindByNamesOrIPv4s(context.Background(), []string{"Test Server"}, []string{"10.0.0.1"})
		assert.NoError(t, err)
		assert.Len(t, servers, 1)
	})
}

func TestGormServerRepository_BatchCreate(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewGormServerRepository(db)

	t.Run("empty", func(t *testing.T) {
		err := repo.BatchCreate(context.Background(), nil)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO .*servers.*`).
			WillReturnRows(sqlmock.NewRows([]string{"server_id"}).AddRow("srv-1"))

		err := repo.BatchCreate(context.Background(), []*domain.Server{
			{ServerName: "Srv 1", IPv4: "10.0.0.1"},
		})
		assert.NoError(t, err)
	})
}

func TestGormServerRepository_Search(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewGormServerRepository(db)

	t.Run("success with filters", func(t *testing.T) {
		mock.ExpectQuery(`SELECT count.*FROM .*servers.* WHERE current_status = \$1 AND \(server_name ILIKE \$2 OR ipv4 ILIKE \$3\)`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

		rows := sqlmock.NewRows([]string{"server_id", "server_name", "ipv4"}).
			AddRow("srv-1", "Test Server", "10.0.0.1")
		mock.ExpectQuery(`SELECT \* FROM .*servers.* WHERE current_status = \$1 AND \(server_name ILIKE \$2 OR ipv4 ILIKE \$3\) ORDER BY server_name desc LIMIT .*`).
			WillReturnRows(rows)

		servers, total, err := repo.Search(context.Background(), repository.ServerListFilter{
			Page:          1,
			PageSize:      10,
			Status:        "ONLINE",
			Name:          "Test",
			SortBy:        "server_name",
			SortDirection: "desc",
		})
		assert.NoError(t, err)
		assert.Len(t, servers, 1)
		assert.Equal(t, int32(10), total)
	})

	t.Run("success with date filters", func(t *testing.T) {
		from, _ := time.Parse("2006-01-02", "2026-01-01")
		to, _ := time.Parse("2006-01-02", "2026-12-31")

		mock.ExpectQuery(`SELECT count.*FROM .*servers.* WHERE created_at >= \$1 AND created_at < \$2`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		rows := sqlmock.NewRows([]string{"server_id", "server_name", "ipv4"}).
			AddRow("srv-2", "Old Server", "10.0.0.2")
		mock.ExpectQuery(`SELECT \* FROM .*servers.* WHERE created_at >= \$1 AND created_at < \$2 ORDER BY created_at desc LIMIT .*`).
			WillReturnRows(rows)

		servers, total, err := repo.Search(context.Background(), repository.ServerListFilter{
			Page:        1,
			PageSize:    10,
			CreatedFrom: from,
			CreatedTo:   to,
		})
		assert.NoError(t, err)
		assert.Len(t, servers, 1)
		assert.Equal(t, int32(5), total)
	})
}

func TestGormServerRepository_Create(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewGormServerRepository(db)

	server := &domain.Server{
		ServerID:      "srv-1",
		ServerName:    " Test Server ",
		IPv4:          " 10.0.0.1 ",
		CurrentStatus: domain.ServerStatusOnline,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO .*servers.*`).
			WillReturnRows(sqlmock.NewRows([]string{"server_id"}).AddRow("srv-1"))

		err := repo.Create(context.Background(), server)
		assert.NoError(t, err)
		assert.Equal(t, "Test Server", server.ServerName)
		assert.Equal(t, "10.0.0.1", server.IPv4)
	})
}

func TestGormServerRepository_Update(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewGormServerRepository(db)

	server := &domain.Server{
		ServerID:      "srv-1",
		ServerName:    "Updated Server",
		IPv4:          "10.0.0.2",
		CurrentStatus: domain.ServerStatusOffline,
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(`UPDATE .*servers.*`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Update(context.Background(), server)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec(`UPDATE .*servers.*`).
			WillReturnResult(sqlmock.NewResult(1, 0)) // 0 rows affected

		err := repo.Update(context.Background(), server)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

func TestGormServerRepository_Delete(t *testing.T) {
	db, mock := setupTestDB(t)
	repo := NewGormServerRepository(db)

	id := "srv-1"

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM .*servers.* WHERE server_id = \$1`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Delete(context.Background(), id)
		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM .*servers.* WHERE server_id = \$1`).
			WillReturnResult(sqlmock.NewResult(1, 0))

		err := repo.Delete(context.Background(), id)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}
