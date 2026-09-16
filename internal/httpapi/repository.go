package httpapi

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// SessionRepository 抽象会话数据访问层。
// 上层（SessionManager）依赖此接口，不依赖具体数据库实现。
// 可以轻松替换为 PostgreSQL、MySQL、内存等实现。
type SessionRepository interface {
	// GetSession 根据 ID 查询会话。不存在时返回 (nil, nil)。
	GetSession(ctx context.Context, id string) (*Session, error)

	// CreateSession 创建新会话。ID 已存在时返回错误。
	CreateSession(ctx context.Context, id string) error

	// TouchSession 更新会话的最后访问时间为当前时间。
	TouchSession(ctx context.Context, id string) error

	// GetMessages 查询会话的消息历史，按时间升序排列，最多返回 limit 条。
	GetMessages(ctx context.Context, sessionID string, limit int) ([]*schema.Message, error)

	// AppendMessages 追加多条消息到会话。
	// 用事务保证多条消息要么都写入，要么都不写入。
	AppendMessages(ctx context.Context, sessionID string, messages ...*schema.Message) error

	// DeleteExpired 删除超过 ttl 未访问的会话，返回删除的会话数。
	// ON DELETE CASCADE 会自动删除关联的消息。
	DeleteExpired(ctx context.Context, ttl time.Duration) (int64, error)
}

// === F.3 新增：内存实现，用于测试 ===

// memoryMessage 是内存存储的消息条目，保留插入顺序。
type memoryMessage struct {
	seq       int // 插入序号，用于排序
	message   *schema.Message
	createdAt time.Time
}

// MemorySessionRepository 是 SessionRepository 的内存实现。
// 用于单元测试，不需要真实数据库，测试速度快。
// 并发安全：所有方法加 mutex。
type MemorySessionRepository struct {
	mu       sync.Mutex
	sessions map[string]*Session         // sessionID -> Session
	messages map[string][]*memoryMessage // sessionID -> 消息列表
	seq      int                         // 全局自增序号，保证消息顺序
}

// NewMemorySessionRepository 创建空的内存存储。
func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{
		sessions: make(map[string]*Session),
		messages: make(map[string][]*memoryMessage),
	}
}

func (r *MemorySessionRepository) GetSession(_ context.Context, id string) (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.sessions[id]
	if !ok {
		return nil, nil // 不存在返回 (nil, nil)，和 PostgreSQL 的 pgx.ErrNoRows 语义一致
	}
	// 返回副本，防止外部修改内部状态
	copy := *s
	return &copy, nil
}

func (r *MemorySessionRepository) CreateSession(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[id]; exists {
		return errSessionExists // 模拟 PostgreSQL 的唯一约束冲突
	}

	r.sessions[id] = &Session{
		ID:         id,
		LastAccess: time.Now(),
	}
	return nil
}

func (r *MemorySessionRepository) TouchSession(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s, ok := r.sessions[id]; ok {
		s.LastAccess = time.Now()
	}
	return nil // 不存在也不报错，和 PostgreSQL 的 UPDATE 行为一致（0 行受影响）
}

func (r *MemorySessionRepository) GetMessages(_ context.Context, sessionID string, limit int) ([]*schema.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	msgs := r.messages[sessionID]
	if len(msgs) == 0 {
		return []*schema.Message{}, nil
	}

	// 按插入序号排序（模拟 ORDER BY created_at ASC）
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].seq < msgs[j].seq
	})

	// 取最新的 limit 条（模拟子查询 DESC LIMIT + 外层 ASC）
	start := len(msgs) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*schema.Message, 0, limit)
	for _, m := range msgs[start:] {
		// 深拷贝消息，防止外部修改内部状态
		data, _ := json.Marshal(m.message)
		var copy schema.Message
		_ = json.Unmarshal(data, &copy)
		result = append(result, &copy)
	}
	return result, nil
}

// SetLastAccess 是测试辅助方法，手动设置会话的最后访问时间。
// 仅用于测试（如模拟过期会话），生产代码不应调用。
func (r *MemorySessionRepository) SetLastAccess(id string, t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s, ok := r.sessions[id]; ok {
		s.LastAccess = t
	}
}

func (r *MemorySessionRepository) AppendMessages(_ context.Context, sessionID string, messages ...*schema.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 事务模拟：要么全部追加，要么都不追加
	// 内存操作不会失败，所以直接追加
	for _, msg := range messages {
		r.seq++
		r.messages[sessionID] = append(r.messages[sessionID], &memoryMessage{
			seq:       r.seq,
			message:   msg,
			createdAt: time.Now(),
		})
	}
	return nil
}

func (r *MemorySessionRepository) DeleteExpired(_ context.Context, ttl time.Duration) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-ttl)
	var deleted int64

	for id, s := range r.sessions {
		if s.LastAccess.Before(cutoff) {
			delete(r.sessions, id)
			delete(r.messages, id) // 模拟 ON DELETE CASCADE
			deleted++
		}
	}
	return deleted, nil
}

// errSessionExists 是内存实现模拟 PostgreSQL 唯一约束冲突的错误。
var errSessionExists = &memoryError{msg: "session already exists"}

type memoryError struct{ msg string }

func (e *memoryError) Error() string { return e.msg }
