package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSessionRepository 是 SessionRepository 的 PostgreSQL 实现。
// 所有 SQL 操作集中在此文件，SessionManager 不直接操作数据库。
type PostgresSessionRepository struct {
	db *pgxpool.Pool
}

// NewPostgresSessionRepository 创建 PostgreSQL 仓储。
func NewPostgresSessionRepository(db *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{db: db}
}

func (r *PostgresSessionRepository) GetSession(ctx context.Context, id string) (*Session, error) {
	var lastAccess time.Time
	err := r.db.QueryRow(ctx, `
		SELECT last_access FROM sessions WHERE id = $1
	`, id).Scan(&lastAccess)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // 不存在返回 (nil, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("get session %q: %w", id, err)
	}
	return &Session{ID: id, LastAccess: lastAccess}, nil
}

func (r *PostgresSessionRepository) CreateSession(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sessions (id, last_access) VALUES ($1, NOW())
	`, id)
	if err != nil {
		return fmt.Errorf("create session %q: %w", id, err)
	}
	return nil
}

func (r *PostgresSessionRepository) TouchSession(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE sessions SET last_access = NOW() WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("touch session %q: %w", id, err)
	}
	return nil
}

func (r *PostgresSessionRepository) GetMessages(ctx context.Context, sessionID string, limit int) ([]*schema.Message, error) {
	// 子查询先倒序取最新的 N 条，外层再正序排列，保证时间顺序
	rows, err := r.db.Query(ctx, `
		SELECT message_json
		FROM (
			SELECT message_json, created_at
			FROM messages
			WHERE session_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		) recent
		ORDER BY created_at ASC
	`, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("query history for session %q: %w", sessionID, err)
	}
	defer rows.Close()

	var messages []*schema.Message
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			log.Printf("scan message for session %q: %v", sessionID, err)
			continue
		}

		var msg schema.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("unmarshal message for session %q: %v", sessionID, err)
			continue
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate history for session %q: %w", sessionID, err)
	}

	if messages == nil {
		messages = []*schema.Message{}
	}
	return messages, nil
}

// AppendMessages 用事务保证多条消息要么都写入，要么都不写入。
// 事务边界在此方法内部，上层不需要关心事务控制。
func (r *PostgresSessionRepository) AppendMessages(ctx context.Context, sessionID string, messages ...*schema.Message) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin append transaction for session %q: %w", sessionID, err)
	}
	defer tx.Rollback(ctx) // 提交后 Rollback 是 no-op

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal message: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO messages (session_id, role, content, message_json)
			VALUES ($1, $2, $3, $4)
		`, sessionID, msg.Role, msg.Content, data)
		if err != nil {
			return fmt.Errorf("insert message to session %q: %w", sessionID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit append transaction for session %q: %w", sessionID, err)
	}
	return nil
}

func (r *PostgresSessionRepository) DeleteExpired(ctx context.Context, ttl time.Duration) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM sessions WHERE last_access < NOW() - $1::interval
	`, ttl.String())
	if err != nil {
		return 0, fmt.Errorf("cleanup expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
