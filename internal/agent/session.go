package agent

import "go-agent/internal/llm"

// Session 代表一次对话会话，保存完整的消息历史
type Session struct {
	ID       string
	Messages []llm.Message
}

type SessionManager struct {
	sessions map[string]*Session
}

func NewSession(id string) *Session {
	return &Session{
		ID:       id,
		Messages: []llm.Message{},
	}
}

// AddMessage 追加一条消息到会话历史
func (s *Session) AddMessage(message llm.Message) {
	s.Messages = append(s.Messages, message)
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// GetOrCreate 获取会话，不存在则创建
func (m *SessionManager) GetOrCreate(id string) *Session {
	if s, ok := m.sessions[id]; ok {
		return s
	}
	s := NewSession(id)
	m.sessions[id] = s
	return s
}
