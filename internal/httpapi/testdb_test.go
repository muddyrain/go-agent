package httpapi

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const testDBConnString = "postgres:///agenthub_test?sslmode=disable"

// getTestDB 返回测试数据库连接池，并确保表结构存在。
// 如果测试数据库不可用，跳过测试（不强制要求所有环境都有数据库）。
func getTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	db, err := pgxpool.New(context.Background(), testDBConnString)
	if err != nil {
		t.Skipf("skipping test: cannot connect to test database: %v", err)
	}

	// 执行建表 SQL，确保表存在
	schemaSQL := `
	CREATE TABLE IF NOT EXISTS sessions (
		id          TEXT PRIMARY KEY,
		last_access TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS messages (
		id           BIGSERIAL PRIMARY KEY,
		session_id   TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
		role         TEXT NOT NULL,
		content      TEXT NOT NULL DEFAULT '',
		message_json JSONB NOT NULL DEFAULT '{}',
		created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	`
	if _, err := db.Exec(context.Background(), schemaSQL); err != nil {
		db.Close()
		t.Skipf("skipping test: cannot create test schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// cleanupTestDB 清空测试数据库中的所有数据。
// TRUNCATE ... CASCADE 会同时清空 sessions 和 messages。
func cleanupTestDB(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	if _, err := db.Exec(context.Background(), "TRUNCATE TABLE sessions CASCADE"); err != nil {
		t.Fatalf("cleanup test database: %v", err)
	}
}
