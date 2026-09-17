package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

// Session 表示一个对话会话，持有该会话的历史消息。
// History 只在 SessionManager 的方法内被修改，外部通过 GetHistory
type Session struct {
	ID         string
	History    []*schema.Message
	LastAccess time.Time
}

// SessionManager 管理会话的业务逻辑。
// 数据访问委托给 SessionRepository 接口，SessionManager 不直接操作数据库。
// 业务逻辑包括：ID 生成、错误降级、清理调度、maxHistory 限制。
type SessionManager struct {
	repo            SessionRepository
	maxHistory      int // 每个 session 最多保留的消息条数
	ttl             time.Duration
	cleanupInterval time.Duration
}

// NewSessionManager 创建会话管理器。
// repo 是 SessionRepository 接口的实现（生产用 PostgresSessionRepository，测试用 MemorySessionRepository）。
func NewSessionManager(repo SessionRepository, maxHistory int, ttl time.Duration, cleanupInterval time.Duration) *SessionManager {
	return &SessionManager{
		repo:            repo,
		maxHistory:      maxHistory,
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
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
		s, err := sm.repo.GetSession(context.Background(), id)
		if err != nil {
			// 数据库出错，记录日志但降级为创建新会话，避免阻塞用户请求
			log.Printf("get session %q: %v", id, err)
		} else if s != nil {
			// 会话存在，更新最后访问时间
			if err := sm.repo.TouchSession(context.Background(), id); err != nil {
				log.Printf("touch session %q: %v", id, err)
			}
			return s
		}
		// s == nil 且 err == nil 表示会话不存在，继续创建新会话
	}

	// 创建新会话：crypto/rand 生成 16 字节随机数，hex 编码为 32 字符
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		id = fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	} else {
		id = hex.EncodeToString(buf)
	}

	if err := sm.repo.CreateSession(context.Background(), id); err != nil {
		log.Printf("create session %q: %v", id, err)
	}

	return &Session{ID: id, LastAccess: time.Now()}
}

// GetHistory 返回指定会话的消息历史。
// 业务逻辑：传递 maxHistory 限制，数据访问委托给 Repository。
func (sm *SessionManager) GetHistory(id string) []*schema.Message {
	messages, err := sm.repo.GetMessages(context.Background(), id, sm.maxHistory)
	if err != nil {
		log.Printf("get history for session %q: %v", id, err)
		return []*schema.Message{}
	}
	return messages
}

// Append 向会话追加一轮对话（用户消息 + 助手消息）。
// 业务逻辑：更新最后访问时间；数据访问（含事务）委托给 Repository。
func (sm *SessionManager) Append(id string, userMsg, assistantMsg *schema.Message) {
	// 事务在 Repository.AppendMessages 内部保证多条消息原子性
	if err := sm.repo.AppendMessages(context.Background(), id, userMsg, assistantMsg); err != nil {
		log.Printf("append messages to session %q: %v", id, err)
		return
	}

	// 更新最后访问时间（不在事务内，失败不影响消息已写入）
	if err := sm.repo.TouchSession(context.Background(), id); err != nil {
		log.Printf("touch session in append %q: %v", id, err)
	}
}

// StartCleanup 启动后台 goroutine，定期清理过期会话。
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
func (sm *SessionManager) cleanupExpired() {
	deleted, err := sm.repo.DeleteExpired(context.Background(), sm.ttl)
	if err != nil {
		log.Printf("cleanup expired sessions: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("cleaned up %d expired sessions", deleted)
	}
}
