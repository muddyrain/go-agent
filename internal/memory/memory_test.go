package memory

import (
	"agenthub/internal/llm"
	"agenthub/internal/tokenizer"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSimpleSliding_Validate(t *testing.T) {
	tests := []struct {
		name    string
		maxKeep int
		wantErr bool
	}{
		{"ok", 5, false},
		{"zero", 0, true},
		{"neg", -3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewSimpleSliding(tt.maxKeep)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, m)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, m)
			}
		})
	}
}

func TestSimpleSliding_Apply(t *testing.T) {
	sysMsg := llm.SystemMessage("你是助手")
	u1 := llm.UserMessage("问题1")
	a1 := llm.AssistantMessage("回答1")
	u2 := llm.UserMessage("问题2")
	a2 := llm.AssistantMessage("回答2")
	u3 := llm.UserMessage("问题3")
	a3 := llm.AssistantMessage("回答3")

	tests := []struct {
		name    string
		maxKeep int
		in      []llm.Message
		want    []llm.Message
	}{
		{
			name:    "empty input",
			maxKeep: 2,
			in:      []llm.Message{},
			want:    []llm.Message{},
		},
		{
			name:    "less than maxKeep with system",
			maxKeep: 10,
			in:      []llm.Message{sysMsg, u1, a1},
			want:    []llm.Message{sysMsg, u1, a1},
		},
		{
			name:    "has system, overflow, keep last 2 non‑system",
			maxKeep: 2,
			in:      []llm.Message{sysMsg, u1, a1, u2, a2, u3, a3},
			// system + [u2,a2,u3,a3]? maxKeep=2代表保留2条非system消息 → u3,a3
			want: []llm.Message{sysMsg, u3, a3},
		},
		{
			name:    "no system overflow",
			maxKeep: 2,
			in:      []llm.Message{u1, a1, u2, a2, u3, a3},
			want:    []llm.Message{u3, a3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem, _ := NewSimpleSliding(tt.maxKeep)
			out, err := mem.Apply(tt.in)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, out)
		})
	}
}

func TestTokenBudgetMemory_Apply(t *testing.T) {
	sys := llm.SystemMessage("你是助手")
	u1 := llm.UserMessage("q1")
	a1 := llm.AssistantMessage("a1")
	u2 := llm.UserMessage("q2")
	a2 := llm.AssistantMessage("a2")
	u3 := llm.UserMessage("q3")

	// fake：每条消息占10 token
	tk := &tokenizer.FakeTokenizer{PerMessage: 10}

	tests := []struct {
		name    string
		budget  int
		in      []llm.Message
		want    []llm.Message
		wantErr bool
	}{
		{
			name:    "fit all",
			budget:  100,
			in:      []llm.Message{sys, u1, a1, u2, a2},
			want:    []llm.Message{sys, u1, a1, u2, a2},
			wantErr: false,
		},
		{
			name:    "overflow drop old",
			budget:  30, // sys(10)+保留后面两条=30
			in:      []llm.Message{sys, u1, a1, u2, a2, u3},
			want:    []llm.Message{sys, a2, u3},
			wantErr: false,
		},
		{
			name:    "system itself over budget",
			budget:  5,
			in:      []llm.Message{sys},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem, err := NewTokenBudgetMemory(tt.budget, tk)
			assert.NoError(t, err)
			out, err := mem.Apply(tt.in)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, out)
			}
		})
	}
}
