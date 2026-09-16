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
	db := getTestDB(t)
	cleanupTestDB(t, db)

	sm := NewSessionManager(NewMemorySessionRepository(), 20)
	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}
	if sm.maxHistory != 20 {
		t.Fatalf("maxHistory = %d, want 20", sm.maxHistory)
	}
}

func TestGetOrCreateWithEmptyIDCreatesNew(t *testing.T) {
	db := getTestDB(t)
	cleanupTestDB(t, db)

	sm := NewSessionManager(NewMemorySessionRepository(), 20)

	s := sm.GetOrCreate("")
	if s == nil {
		t.Fatal("GetOrCreate returned nil")
	}
	if s.ID == "" {
		t.Fatal("new session has empty ID")
	}

	// 再次调用空 ID 应该创建另一个新会话
	s2 := sm.GetOrCreate("")
	if s2.ID == s.ID {
		t.Fatal("two calls with empty ID returned the same session")
	}
}

func TestGetOrCreateReturnsExisting(t *testing.T) {
	db := getTestDB(t)
	cleanupTestDB(t, db)

	sm := NewSessionManager(NewMemorySessionRepository(), 20)

	s1 := sm.GetOrCreate("")
	s2 := sm.GetOrCreate(s1.ID)

	if s1.ID != s2.ID {
		t.Fatalf("GetOrCreate with existing ID returned different session: %s vs %s", s1.ID, s2.ID)
	}
}

func TestGetOrCreateWithUnknownIDCreatesNew(t *testing.T) {
	db := getTestDB(t)
	cleanupTestDB(t, db)

	sm := NewSessionManager(NewMemorySessionRepository(), 20)

	// 传入一个不存在的 ID，应该创建新会话（忽略传入的 ID，生成新 ID）
	s := sm.GetOrCreate("nonexistent-id-12345")
	if s == nil {
		t.Fatal("GetOrCreate returned nil")
	}
	if s.ID == "nonexistent-id-12345" {
		t.Fatal("new session should not use the nonexistent input ID")
	}
}

func TestGetHistoryReturnsCopy(t *testing.T) {
	db := getTestDB(t)
	cleanupTestDB(t, db)

	sm := NewSessionManager(NewMemorySessionRepository(), 20)
	s := sm.GetOrCreate("")

	userMsg := schema.UserMessage("hello")
	assistantMsg := schema.AssistantMessage("hi", nil)
	sm.Append(s.ID, userMsg, assistantMsg)

	history := sm.GetHistory(s.ID)
	if len(history) != 2 {
		t.Fatalf("GetHistory returned %d messages, want 2", len(history))
	}

	// 修改返回的切片元素，不应影响内部状态
	history[0] = schema.UserMessage("modified")

	internal := sm.GetHistory(s.ID)
	if internal[0].Content == "modified" {
		t.Fatal("modifying returned copy affected internal state")
	}
	if internal[0].Content != "hello" {
		t.Fatalf("internal history changed: got %q, want %q", internal[0].Content, "hello")
	}
}

func TestGetHistoryUnknownIDReturnsEmpty(t *testing.T) {
	db := getTestDB(t)
	cleanupTestDB(t, db)

	sm := NewSessionManager(NewMemorySessionRepository(), 20)
	history := sm.GetHistory("nonexistent")
	// PostgreSQL 版本返回空切片而不是 nil
	if len(history) != 0 {
		t.Fatalf("GetHistory for unknown ID returned %d messages, want 0", len(history))
	}
}

func TestAppendAddsMessages(t *testing.T) {
	db := getTestDB(t)
	cleanupTestDB(t, db)

	sm := NewSessionManager(NewMemorySessionRepository(), 20)
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
	db := getTestDB(t)
	cleanupTestDB(t, db)

	// maxHistory = 4，追加 3 轮（6 条）后应该只保留最近 4 条
	sm := NewSessionManager(NewMemorySessionRepository(), 4)
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
	// 应该保留第 1 轮和第 2 轮（q1,a1,q2,a2），丢弃第 0 轮
	if history[0].Content != "q1" {
		t.Fatalf("first retained message = %q, want q1", history[0].Content)
	}
	if history[3].Content != "a2" {
		t.Fatalf("last retained message = %q, want a2", history[3].Content)
	}
}

func TestAppendUnknownIDDoesNotPanic(t *testing.T) {
	db := getTestDB(t)
	cleanupTestDB(t, db)

	sm := NewSessionManager(NewMemorySessionRepository(), 20)
	// 对不存在的 ID 调用 Append 应该安全返回，不 panic
	// （PostgreSQL 外键约束会导致 INSERT 失败，Append 内部记录日志后返回）
	sm.Append("nonexistent", schema.UserMessage("x"), schema.AssistantMessage("y", nil))
}

func TestGenerateSessionIDUnique(t *testing.T) {
	db := getTestDB(t)
	cleanupTestDB(t, db)

	// 通过 GetOrCreate 生成 100 个 ID，不应有重复
	sm := NewSessionManager(NewMemorySessionRepository(), 20)
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s := sm.GetOrCreate("")
		if seen[s.ID] {
			t.Fatalf("duplicate session ID generated: %s", s.ID)
		}
		seen[s.ID] = true
		if len(s.ID) != 32 {
			t.Fatalf("session ID length = %d, want 32", len(s.ID))
		}
	}
}

// --- 并发安全与过期清理测试 ---

func TestSessionManagerConcurrentAccess(t *testing.T) {
	db := getTestDB(t)
	cleanupTestDB(t, db)

	sm := NewSessionManager(NewMemorySessionRepository(), 20)
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
	repo := NewMemorySessionRepository()
	sm := NewSessionManager(repo, 20)

	// 创建一个正常 session（last_access = 当前时间，未过期）
	active := sm.GetOrCreate("")

	// 创建一个 session，然后用内存实现的测试辅助方法把 last_access 设为 1 小时前（已过期）
	expired := sm.GetOrCreate("")
	repo.SetLastAccess(expired.ID, time.Now().Add(-1*time.Hour))

	// 执行清理
	sm.cleanupExpired()

	// 验证：active 还在，expired 被删除
	// 通过 Repository 接口验证，不用 SQL
	activeSession, err := repo.GetSession(context.Background(), active.ID)
	if err != nil {
		t.Fatalf("get active session: %v", err)
	}
	if activeSession == nil {
		t.Fatal("active session should not be cleaned up")
	}

	expiredSession, err := repo.GetSession(context.Background(), expired.ID)
	if err != nil {
		t.Fatalf("get expired session: %v", err)
	}
	if expiredSession != nil {
		t.Fatal("expired session should be cleaned up")
	}
}

func TestSessionManagerStartCleanupStopsOnCancel(t *testing.T) {
	sm := NewSessionManager(NewMemorySessionRepository(), 20)

	ctx, cancel := context.WithCancel(context.Background())
	sm.StartCleanup(ctx)

	// 取消 context，goroutine 应退出
	cancel()

	// 给 goroutine 一点时间退出
	time.Sleep(50 * time.Millisecond)

	// cancel 后清理 goroutine 不应再影响数据（这里只是验证不 panic）
	sm.GetOrCreate("")
}
