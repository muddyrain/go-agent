package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 默认会话过期时间与清理间隔。
// 会话 30 分钟无访问则过期；每 5 分钟检查一次。
const (
	defaultSessionTTL      = 30 * time.Minute
	defaultCleanupInterval = 5 * time.Minute
)

// Session 表示一个对话会话，持有该会话的历史消息。
// History 只在 SessionManager 的方法内被修改，外部通过 GetHistory
type Session struct {
	ID         string
	History    []*schema.Message
	LastAccess time.Time
}

// SessionManager 管理持久化到 PostgreSQL 的会话存储。
// 并发安全由数据库事务和行锁保证，不需要应用层 mutex。
type SessionManager struct {
	db              *pgxpool.Pool
	maxHistory      int // 每个 session 最多保留的消息条数
	ttl             time.Duration
	cleanupInterval time.Duration
}

// NewSessionManager 创建会话管理器。
// db 是已建立的 PostgreSQL 连接池；maxHistory 限制每个会话保留的消息条数。
func NewSessionManager(db *pgxpool.Pool, maxHistory int) *SessionManager {
	return &SessionManager{
		db:              db,
		maxHistory:      maxHistory,
		ttl:             defaultSessionTTL,
		cleanupInterval: defaultCleanupInterval,
	}
}

// generateSessionID 生成 16 字节随机数并 hex 编码为 32 字符字符串。
// crypto/rand 是加密安全随机源，比 math/rand 更适合做会话 ID。
func generateSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // 错误忽略：crypto/rand.Read 在 Linux/macOS 上不会失败
	return hex.EncodeToString(b)
}

// GetOrCreate 根据 ID 获取会话；ID 为空或不存在时创建新会话。
// 返回的 Session 只包含 ID 和元数据，历史消息通过 GetHistory 单独获取。
func (sm *SessionManager) GetOrCreate(id string) *Session {
	id = strings.TrimSpace(id)

	if id != "" {
		var lastAccess time.Time

		err := sm.db.QueryRow(context.Background(), `
			SELECT last_access FROM sessions WHERE id = $1
		`, id).Scan(&lastAccess)

		if err == nil {
			// 会话存在，更新最后访问时间
			sm.touchSession(id)
			return &Session{ID: id, LastAccess: lastAccess}
		}

		if !errors.Is(err, pgx.ErrNoRows) {
			// 数据库出错，记录日志但降级为创建新会话，避免阻塞用户请求
			log.Printf("get session %q: %v", id, err)
		}
	}

	// 创建新会话：crypto/rand 生成 16 字节随机数，hex 编码为 32 字符
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// rand.Read 失败极罕见，降级用时间戳保证不崩溃
		id = fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	} else {
		id = hex.EncodeToString(buf)
	}

	_, err := sm.db.Exec(context.Background(), `
		INSERT INTO sessions (id, last_access) VALUES ($1, NOW())
	`, id)
	if err != nil {
		log.Printf("create session %q: %v", id, err)
	}

	return &Session{ID: id, LastAccess: time.Now()}
}

// touchSession 更新会话的最后访问时间。
func (sm *SessionManager) touchSession(id string) {
	_, err := sm.db.Exec(context.Background(), `
		UPDATE sessions SET last_access = NOW() WHERE id = $1
	`, id)
	if err != nil {
		log.Printf("touch session %q: %v", id, err)
	}
}

// GetHistory 返回指定会话的消息历史。
// 按 created_at 升序排列，最多返回 maxHistory 条。
func (sm *SessionManager) GetHistory(id string) []*schema.Message {
	// 子查询先倒序取最新的 N 条，外层再正序排列，保证时间顺序
	rows, err := sm.db.Query(context.Background(), `
		SELECT message_json
		FROM (
			SELECT message_json, created_at
			FROM messages
			WHERE session_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		) recent
		ORDER BY created_at ASC
	`, id, sm.maxHistory)
	if err != nil {
		log.Printf("query history for session %q: %v", id, err)
		return []*schema.Message{}
	}
	defer rows.Close()

	var messages []*schema.Message
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			log.Printf("scan message for session %q: %v", id, err)
			continue
		}

		var msg schema.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("unmarshal message for session %q: %v", id, err)
			continue
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate history for session %q: %v", id, err)
	}

	return messages
}

// Append 向会话追加一轮对话（用户消息 + 助手消息）。
// 用事务保证两条消息要么都写入，要么都不写入。
func (sm *SessionManager) Append(id string, userMsg, assistantMsg *schema.Message) {
	tx, err := sm.db.Begin(context.Background())
	if err != nil {
		log.Printf("begin append transaction for session %q: %v", id, err)
		return
	}
	defer tx.Rollback(context.Background()) // 提交后 Rollback 是 no-op

	if err := sm.insertMessage(tx, id, userMsg); err != nil {
		log.Printf("append user message to session %q: %v", id, err)
		return
	}
	if err := sm.insertMessage(tx, id, assistantMsg); err != nil {
		log.Printf("append assistant message to session %q: %v", id, err)
		return
	}

	// 更新最后访问时间
	if _, err := tx.Exec(context.Background(), `
		UPDATE sessions SET last_access = NOW() WHERE id = $1
	`, id); err != nil {
		log.Printf("touch session in append %q: %v", id, err)
		return
	}

	if err := tx.Commit(context.Background()); err != nil {
		log.Printf("commit append transaction for session %q: %v", id, err)
	}
}

// insertMessage 在事务中插入一条消息。
func (sm *SessionManager) insertMessage(tx pgx.Tx, sessionID string, msg *schema.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	_, err = tx.Exec(context.Background(), `
		INSERT INTO messages (session_id, role, content, message_json)
		VALUES ($1, $2, $3, $4)
	`, sessionID, msg.Role, msg.Content, data)
	return err
}

// StartCleanup 启动后台 goroutine，定期清理过期会话。
// ctx 被取消时 goroutine 自动退出；调用方应在服务关闭时 cancel ctx。
func (sm *SessionManager) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(sm.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sm.cleanupExpired()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// cleanupExpired 删除超过 TTL 未访问的会话。
// ON DELETE CASCADE 会自动删除关联的消息。
func (sm *SessionManager) cleanupExpired() {
	tag, err := sm.db.Exec(context.Background(), `
		DELETE FROM sessions WHERE last_access < NOW() - $1::interval
	`, sm.ttl.String())

	if err != nil {
		log.Printf("cleanup expired sessions: %v", err)
		return
	}
	if tag.RowsAffected() > 0 {
		log.Printf("cleaned up %d expired sessions", tag.RowsAffected())
	}
}
