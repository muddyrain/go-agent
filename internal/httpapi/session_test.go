package httpapi

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

// --- SessionManager 单元测试 ---

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager(20)
	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}
	if sm.maxHistory != 20 {
		t.Fatalf("maxHistory = %d, want 20", sm.maxHistory)
	}
	if sm.sessions == nil {
		t.Fatal("sessions map is nil")
	}
}

func TestGetOrCreateWithEmptyIDCreatesNew(t *testing.T) {
	sm := NewSessionManager(20)

	s := sm.GetOrCreate("")
	if s == nil {
		t.Fatal("GetOrCreate returned nil")
	}
	if s.ID == "" {
		t.Fatal("new session has empty ID")
	}
	if len(s.History) != 0 {
		t.Fatalf("new session has %d history messages, want 0", len(s.History))
	}

	// 再次调用空 ID 应该创建另一个新会话
	s2 := sm.GetOrCreate("")
	if s2.ID == s.ID {
		t.Fatal("two calls with empty ID returned the same session")
	}
}

func TestGetOrCreateReturnsExisting(t *testing.T) {
	sm := NewSessionManager(20)

	s1 := sm.GetOrCreate("")
	s2 := sm.GetOrCreate(s1.ID)

	if s1.ID != s2.ID {
		t.Fatalf("GetOrCreate with existing ID returned different session: %s vs %s", s1.ID, s2.ID)
	}
}

func TestGetOrCreateWithUnknownIDCreatesNew(t *testing.T) {
	sm := NewSessionManager(20)

	// 传入一个不存在的 ID，应该创建新会话（忽略传入的 ID）
	s := sm.GetOrCreate("nonexistent-id-12345")
	if s == nil {
		t.Fatal("GetOrCreate returned nil")
	}
	if s.ID == "nonexistent-id-12345" {
		t.Fatal("new session should not use the nonexistent input ID")
	}
}

func TestGetHistoryReturnsCopy(t *testing.T) {
	sm := NewSessionManager(20)
	s := sm.GetOrCreate("")

	userMsg := schema.UserMessage("hello")
	assistantMsg := schema.AssistantMessage("hi", nil)
	sm.Append(s.ID, userMsg, assistantMsg)

	history := sm.GetHistory(s.ID)
	if len(history) != 2 {
		t.Fatalf("GetHistory returned %d messages, want 2", len(history))
	}

	// 修改返回的副本，不应影响内部状态
	history[0] = schema.UserMessage("modified")

	internal := sm.GetHistory(s.ID)
	if internal[0].Content == "modified" {
		t.Fatal("modifying returned copy affected internal state")
	}
	if internal[0].Content != "hello" {
		t.Fatalf("internal history changed: got %q, want %q", internal[0].Content, "hello")
	}
}

func TestGetHistoryUnknownIDReturnsNil(t *testing.T) {
	sm := NewSessionManager(20)
	history := sm.GetHistory("nonexistent")
	if history != nil {
		t.Fatalf("GetHistory for unknown ID returned %v, want nil", history)
	}
}

func TestAppendAddsMessages(t *testing.T) {
	sm := NewSessionManager(20)
	s := sm.GetOrCreate("")

	userMsg := schema.UserMessage("question")
	assistantMsg := schema.AssistantMessage("answer", nil)
	sm.Append(s.ID, userMsg, assistantMsg)

	history := sm.GetHistory(s.ID)
	if len(history) != 2 {
		t.Fatalf("after Append, history has %d messages, want 2", len(history))
	}
	if history[0].Role != schema.User || history[0].Content != "question" {
		t.Fatalf("first message = %+v, want user/question", history[0])
	}
	if history[1].Role != schema.Assistant || history[1].Content != "answer" {
		t.Fatalf("second message = %+v, want assistant/answer", history[1])
	}
}

func TestAppendTruncatesOldMessages(t *testing.T) {
	// maxHistory = 4，追加 3 轮（6 条）后应该只保留最近 4 条
	sm := NewSessionManager(4)
	s := sm.GetOrCreate("")

	for i := 0; i < 3; i++ {
		userMsg := schema.UserMessage("q" + string(rune('0'+i)))
		assistantMsg := schema.AssistantMessage("a"+string(rune('0'+i)), nil)
		sm.Append(s.ID, userMsg, assistantMsg)
	}

	history := sm.GetHistory(s.ID)
	if len(history) != 4 {
		t.Fatalf("after truncation, history has %d messages, want 4", len(history))
	}
	// 应该保留第 2 轮和第 3 轮（q1,a1,q2,a2），丢弃第 0 轮和第 1 轮
	if history[0].Content != "q1" {
		t.Fatalf("first retained message = %q, want q1", history[0].Content)
	}
	if history[3].Content != "a2" {
		t.Fatalf("last retained message = %q, want a2", history[3].Content)
	}
}

func TestAppendUnknownIDDoesNotPanic(t *testing.T) {
	sm := NewSessionManager(20)
	// 对不存在的 ID 调用 Append 应该安全返回，不 panic
	sm.Append("nonexistent", schema.UserMessage("x"), schema.AssistantMessage("y", nil))
}

func TestGenerateSessionIDFormat(t *testing.T) {
	id := generateSessionID()
	// 16 字节 hex 编码 = 32 字符
	if len(id) != 32 {
		t.Fatalf("session ID length = %d, want 32", len(id))
	}
	// 应该只包含 hex 字符
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("session ID contains non-hex character: %q", c)
		}
	}
}

func TestGenerateSessionIDUnique(t *testing.T) {
	// 生成 100 个 ID，不应有重复
	sm := NewSessionManager(20)
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s := sm.GetOrCreate("")
		if seen[s.ID] {
			t.Fatalf("duplicate session ID generated: %s", s.ID)
		}
		seen[s.ID] = true
	}
}

// --- 并发安全与过期清理测试 ---

func TestSessionManagerConcurrentAccess(t *testing.T) {
	sm := NewSessionManager(20)
	session := sm.GetOrCreate("")

	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 并发读写同一个 session
				sm.GetHistory(session.ID)
				sm.Append(
					session.ID,
					schema.UserMessage("q"),
					schema.AssistantMessage("a", nil),
				)
				// 并发创建新 session
				sm.GetOrCreate("")
			}
		}()
	}

	wg.Wait()

	// 验证历史消息数量正确：10 goroutine × 100 次 × 2 条 = 2000 条，
	// 但 maxHistory=20，所以应该被截断到 20 条
	history := sm.GetHistory(session.ID)
	if len(history) != 20 {
		t.Fatalf("history length = %d, want 20 (truncated by maxHistory)", len(history))
	}
}

func TestSessionManagerCleanupExpired(t *testing.T) {
	sm := NewSessionManager(20)

	// 创建一个正常 session
	active := sm.GetOrCreate("")

	// 创建一个 session 并手动把 LastAccess 设为 1 小时前（已过期）
	expired := sm.GetOrCreate("")
	sm.mu.Lock()
	expired.LastAccess = time.Now().Add(-1 * time.Hour)
	sm.mu.Unlock()

	// 执行清理
	sm.cleanupExpired()

	// 验证：active 还在，expired 被删除
	if sm.GetHistory(active.ID) == nil {
		t.Fatal("active session should not be cleaned up")
	}
	if sm.GetHistory(expired.ID) != nil {
		t.Fatal("expired session should be cleaned up")
	}
}

func TestSessionManagerStartCleanup(t *testing.T) {
	// 用极短的 TTL 和清理间隔测试 StartCleanup
	sm := &SessionManager{
		sessions:        make(map[string]*Session),
		maxHistory:      20,
		ttl:             50 * time.Millisecond,
		cleanupInterval: 20 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sm.StartCleanup(ctx)

	// 创建一个 session
	s := sm.GetOrCreate("")
	sessionID := s.ID

	// 验证创建后存在
	if sm.GetHistory(sessionID) == nil {
		t.Fatal("session should exist after creation")
	}

	// 等待超过 TTL，让后台清理 goroutine 执行
	time.Sleep(150 * time.Millisecond)

	// 验证已被清理
	if sm.GetHistory(sessionID) != nil {
		t.Fatal("session should be cleaned up after TTL")
	}
}

func TestSessionManagerStartCleanupStopsOnCancel(t *testing.T) {
	sm := &SessionManager{
		sessions:        make(map[string]*Session),
		maxHistory:      20,
		ttl:             time.Hour,
		cleanupInterval: 10 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	sm.StartCleanup(ctx)

	// 取消 context，goroutine 应退出
	cancel()

	// 给 goroutine 一点时间退出（无法直接检测 goroutine 是否退出，
	// 但如果有 goroutine 泄漏，race detector 或长时间运行测试会发现）
	time.Sleep(50 * time.Millisecond)

	// cancel 后清理 goroutine 不应再影响数据（这里只是验证不 panic）
	sm.GetOrCreate("")
}
