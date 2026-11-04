package main

import "sync"

// sessionStore 只保存每个会话的成功处理次数；通过构造函数初始化，不复制。
type sessionStore struct {
	mu    sync.Mutex
	turns map[string]int
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		turns: make(map[string]int),
	}
}

func (s *sessionStore) nextTurn(sessionID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.turns[sessionID]++
	return s.turns[sessionID]
}
