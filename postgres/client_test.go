package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	dbName = "testdb"
	dbUser = "testuser"
	dbPass = "testpass"
)

func setupPostgres(t *testing.T) (*Client, func()) {
	ctx := context.Background()

	// PostgreSQL 컨테이너 시작
	postgresContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPass),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	// 연결 문자열 얻기
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// 클라이언트 생성
	client, err := NewClient(connStr)
	require.NoError(t, err)

	// 연결 확인
	err = client.Ping(ctx)
	require.NoError(t, err)

	cleanup := func() {
		client.Close()
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}

	return client, cleanup
}

func TestPostgreSQLCreateTable(t *testing.T) {
	client, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	tableName := "users"

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	assert.NoError(t, err)

	// 테이블 존재 확인
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = $1
		)
	`
	err = client.db.QueryRowContext(ctx, query, tableName).Scan(&exists)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestPostgreSQLInsertAndGet(t *testing.T) {
	client, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	tableName := "users"

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 사용자 추가
	id, err := client.InsertUser(ctx, tableName, "John Doe", "john@example.com")
	assert.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// 사용자 조회
	user, err := client.GetUser(ctx, tableName, id)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
}

func TestPostgreSQLUpdate(t *testing.T) {
	client, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	tableName := "users"

	// 테이블 생성 및 사용자 추가
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	id, err := client.InsertUser(ctx, tableName, "John Doe", "john@example.com")
	require.NoError(t, err)

	// 사용자 업데이트
	err = client.UpdateUser(ctx, tableName, id, "Jane Doe", "jane@example.com")
	assert.NoError(t, err)

	// 업데이트 확인
	user, err := client.GetUser(ctx, tableName, id)
	assert.NoError(t, err)
	assert.Equal(t, "Jane Doe", user.Name)
	assert.Equal(t, "jane@example.com", user.Email)
}

func TestPostgreSQLDelete(t *testing.T) {
	client, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	tableName := "users"

	// 테이블 생성 및 사용자 추가
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	id, err := client.InsertUser(ctx, tableName, "John Doe", "john@example.com")
	require.NoError(t, err)

	// 사용자 삭제
	err = client.DeleteUser(ctx, tableName, id)
	assert.NoError(t, err)

	// 삭제 확인
	user, err := client.GetUser(ctx, tableName, id)
	assert.NoError(t, err)
	assert.Nil(t, user)
}

func TestPostgreSQLGetAll(t *testing.T) {
	client, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	tableName := "users"

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 여러 사용자 추가
	users := []struct {
		name  string
		email string
	}{
		{"John Doe", "john@example.com"},
		{"Jane Smith", "jane@example.com"},
		{"Bob Johnson", "bob@example.com"},
	}

	for _, u := range users {
		_, err := client.InsertUser(ctx, tableName, u.name, u.email)
		require.NoError(t, err)
	}

	// 모든 사용자 조회
	allUsers, err := client.GetAllUsers(ctx, tableName)
	assert.NoError(t, err)
	assert.Len(t, allUsers, 3)
}

func TestPostgreSQLWhereClause(t *testing.T) {
	client, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	tableName := "users"

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 여러 사용자 추가
	users := []struct {
		name  string
		email string
	}{
		{"John Doe", "john@example.com"},
		{"John Smith", "john.smith@example.com"},
		{"Jane Doe", "jane@example.com"},
	}

	for _, u := range users {
		_, err := client.InsertUser(ctx, tableName, u.name, u.email)
		require.NoError(t, err)
	}

	// 패턴 검색
	results, err := client.GetUsersByNamePattern(ctx, tableName, "John%")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestPostgreSQLTransaction(t *testing.T) {
	client, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	tableName := "users"

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 트랜잭션 성공 케이스
	err = client.ExecuteInTransaction(ctx, func(tx *sqlx.Tx) error {
		query := fmt.Sprintf(
			"INSERT INTO %s (name, email) VALUES ($1, $2)",
			tableName,
		)
		_, err := tx.ExecContext(ctx, query, "John Doe", "john@example.com")
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, query, "Jane Doe", "jane@example.com")
		return err
	})
	assert.NoError(t, err)

	// 추가된 사용자 확인
	users, err := client.GetAllUsers(ctx, tableName)
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// 트랜잭션 롤백 케이스
	err = client.ExecuteInTransaction(ctx, func(tx *sqlx.Tx) error {
		query := fmt.Sprintf(
			"INSERT INTO %s (name, email) VALUES ($1, $2)",
			tableName,
		)
		_, err := tx.ExecContext(ctx, query, "Bob Johnson", "bob@example.com")
		if err != nil {
			return err
		}

		// 중복 이메일로 인한 에러 발생
		_, err = tx.ExecContext(ctx, query, "Another User", "john@example.com")
		return err
	})
	assert.Error(t, err)

	// 롤백 확인 - 여전히 2명만 있어야 함
	users, err = client.GetAllUsers(ctx, tableName)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestPostgreSQLDropTable(t *testing.T) {
	client, cleanup := setupPostgres(t)
	defer cleanup()

	ctx := context.Background()
	tableName := "users"

	// 테이블 생성
	err := client.CreateTable(ctx, tableName)
	require.NoError(t, err)

	// 테이블 삭제
	err = client.DropTable(ctx, tableName)
	assert.NoError(t, err)

	// 테이블 존재 확인
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = $1
		)
	`
	err = client.db.QueryRowContext(ctx, query, tableName).Scan(&exists)
	assert.NoError(t, err)
	assert.False(t, exists)
}
