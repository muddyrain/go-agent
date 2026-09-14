package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// Session 表示一个对话会话，持有该会话的历史消息。
// History 只在 SessionManager 的方法内被修改，外部通过 GetHistory
// 拿到副本，避免并发读写。
type Session struct {
	ID         string
	History    []*schema.Message
	LastAccess time.Time
}

// SessionManager 管理所有会话的创建、历史读取与追加。
// 使用 sync.Mutex 保证 map 的并发安全：HTTP server 每个请求在独立
// goroutine 中运行，多个请求可能同时读写同一个 session。
type SessionManager struct {
	mu         sync.Mutex
	sessions   map[string]*Session
	maxHistory int // 每个 session 最多保留的消息条数
}

// NewSessionManager 创建会话管理器。
// maxHistory 为每个会话保留的最大消息条数；超过时截断最旧的消息。
func NewSessionManager(maxHistory int) *SessionManager {
	return &SessionManager{
		sessions:   make(map[string]*Session),
		maxHistory: maxHistory,
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
// 调用方拿到的是 *Session 指针，但不应直接修改 History 字段，
// 应通过 Append 方法追加。
func (sm *SessionManager) GetOrCreate(id string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if id != "" {
		if s, ok := sm.sessions[id]; ok {
			s.LastAccess = time.Now()
			return s
		}
	}
	newID := generateSessionID()
	s := &Session{
		ID:         newID,
		History:    nil,
		LastAccess: time.Now(),
	}
	sm.sessions[newID] = s
	return s
}

// GetHistory 返回指定会话的历史消息副本。
// 返回副本是为了防止调用方修改内部切片导致数据竞争。
func (sm *SessionManager) GetHistory(id string) []*schema.Message {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return nil
	}
	// 复制切片头部，Message 指针本身不复制（只读使用）
	history := make([]*schema.Message, len(s.History))
	copy(history, s.History)
	return history
}

// Append 向指定会话追加一轮对话（用户消息 + 助手消息）。
// 超过 maxHistory 时截断最旧的消息，保留最近的 maxHistory 条。
func (sm *SessionManager) Append(id string, userMsg, assistantMsg *schema.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[id]
	if !ok {
		return
	}

	s.History = append(s.History, userMsg, assistantMsg)

	// 超过上限时截断最旧的消息
	if len(s.History) > sm.maxHistory {
		s.History = s.History[len(s.History)-sm.maxHistory:]
	}
	s.LastAccess = time.Now()
}
