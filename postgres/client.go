package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Client는 PostgreSQL 클라이언트를 래핑합니다
type Client struct {
	db *sqlx.DB
}

// User는 사용자 정보를 나타냅니다
type User struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	CreatedAt string    `db:"created_at"`
}

// NewClient는 새로운 PostgreSQL 클라이언트를 생성합니다
func NewClient(connStr string) (*Client, error) {
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, err
	}

	return &Client{db: db}, nil
}

// Close는 데이터베이스 연결을 종료합니다
func (c *Client) Close() error {
	return c.db.Close()
}

// Ping은 데이터베이스 연결을 확인합니다
func (c *Client) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// CreateTable은 테이블을 생성합니다
func (c *Client) CreateTable(ctx context.Context, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, tableName)

	_, err := c.db.ExecContext(ctx, query)
	return err
}

// DropTable은 테이블을 삭제합니다
func (c *Client) DropTable(ctx context.Context, tableName string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	_, err := c.db.ExecContext(ctx, query)
	return err
}

// InsertUser는 사용자를 추가합니다
func (c *Client) InsertUser(ctx context.Context, tableName, name, email string) (int64, error) {
	query := fmt.Sprintf(
		"INSERT INTO %s (name, email) VALUES ($1, $2) RETURNING id",
		tableName,
	)

	var id int64
	err := c.db.QueryRowContext(ctx, query, name, email).Scan(&id)
	return id, err
}

// GetUser는 ID로 사용자를 조회합니다
func (c *Client) GetUser(ctx context.Context, tableName string, id int64) (*User, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", tableName)

	var user User
	err := c.db.GetContext(ctx, &user, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAllUsers는 모든 사용자를 조회합니다
func (c *Client) GetAllUsers(ctx context.Context, tableName string) ([]User, error) {
	query := fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName)

	var users []User
	err := c.db.SelectContext(ctx, &users, query)
	return users, err
}

// UpdateUser는 사용자 정보를 업데이트합니다
func (c *Client) UpdateUser(ctx context.Context, tableName string, id int64, name, email string) error {
	query := fmt.Sprintf(
		"UPDATE %s SET name = $1, email = $2 WHERE id = $3",
		tableName,
	)

	result, err := c.db.ExecContext(ctx, query, name, email, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %d not found", id)
	}

	return nil
}

// DeleteUser는 사용자를 삭제합니다
func (c *Client) DeleteUser(ctx context.Context, tableName string, id int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", tableName)

	result, err := c.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %d not found", id)
	}

	return nil
}

// GetUsersByNamePattern은 이름 패턴으로 사용자를 조회합니다
func (c *Client) GetUsersByNamePattern(ctx context.Context, tableName, pattern string) ([]User, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE name LIKE $1 ORDER BY id", tableName)

	var users []User
	err := c.db.SelectContext(ctx, &users, query, pattern)
	return users, err
}

// BeginTransaction은 트랜잭션을 시작합니다
func (c *Client) BeginTransaction(ctx context.Context) (*sqlx.Tx, error) {
	return c.db.BeginTxx(ctx, nil)
}

// ExecuteInTransaction은 트랜잭션 내에서 함수를 실행합니다
func (c *Client) ExecuteInTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := c.BeginTransaction(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
