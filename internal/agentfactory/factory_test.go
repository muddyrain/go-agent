package agentfactory

import (
	"context"
	"strings"
	"testing"

	"agenthub/internal/llm"
	"agenthub/internal/tokenizer"
	"agenthub/internal/tool"
)

type stubModel struct {
	response llm.Response
	err      error
}

func (m *stubModel) Generate(
	_ context.Context,
	_ llm.Request,
) (llm.Response, error) {
	return m.response, m.err
}

var _ llm.Model = (*stubModel)(nil)

func TestBuildMemorySliding(t *testing.T) {
	mem, err := buildMemory(
		MemoryConfig{
			Type:        MemoryTypeSliding,
			MaxMessages: 2,
		},
		nil,
	)
	if err != nil {
		t.Fatalf(
			"buildMemory() returned error: %v",
			err,
		)
	}

	got, err := mem.Apply(
		[]llm.Message{
			llm.UserMessage("one"),
			llm.AssistantMessage("two"),
			llm.UserMessage("three"),
		},
	)
	if err != nil {
		t.Fatalf(
			"Apply() returned error: %v",
			err,
		)
	}

	if gotLen, wantLen := len(got), 2; gotLen != wantLen {
		t.Fatalf(
			"len(Apply()) = %d, want %d",
			gotLen,
			wantLen,
		)
	}

	if gotContent, wantContent := got[0].Content, "two"; gotContent != wantContent {
		t.Fatalf(
			"first retained message = %q, want %q",
			gotContent,
			wantContent,
		)
	}
}

func TestBuildMemoryTokenBudget(t *testing.T) {
	mem, err := buildMemory(
		MemoryConfig{
			Type:        MemoryTypeTokenBudget,
			TokenBudget: 20,
		},
		&tokenizer.FakeTokenizer{
			PerMessage: 10,
		},
	)
	if err != nil {
		t.Fatalf(
			"buildMemory() returned error: %v",
			err,
		)
	}

	got, err := mem.Apply(
		[]llm.Message{
			llm.UserMessage("one"),
			llm.AssistantMessage("two"),
			llm.UserMessage("three"),
		},
	)
	if err != nil {
		t.Fatalf(
			"Apply() returned error: %v",
			err,
		)
	}

	if gotLen, wantLen := len(got), 2; gotLen != wantLen {
		t.Fatalf(
			"len(Apply()) = %d, want %d",
			gotLen,
			wantLen,
		)
	}

	if gotContent, wantContent := got[0].Content, "two"; gotContent != wantContent {
		t.Fatalf(
			"first retained message = %q, want %q",
			gotContent,
			wantContent,
		)
	}
}

func TestBuildMemoryErrors(t *testing.T) {
	tests := []struct {
		name           string
		cfg            MemoryConfig
		tokenizerValue tokenizer.Tokenizer
		wantError      string
	}{
		{
			name: "sliding max messages is zero",
			cfg: MemoryConfig{
				Type:        MemoryTypeSliding,
				MaxMessages: 0,
			},
			tokenizerValue: nil,
			wantError:      "create sliding memory",
		},
		{
			name: "token budget tokenizer is missing",
			cfg: MemoryConfig{
				Type:        MemoryTypeTokenBudget,
				TokenBudget: 20,
			},
			tokenizerValue: nil,
			wantError:      "tokenizer is required",
		},
		{
			name: "memory type is unsupported",
			cfg: MemoryConfig{
				Type: MemoryType("unknown"),
			},
			tokenizerValue: nil,
			wantError:      "unsupported memory type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildMemory(
				tt.cfg,
				tt.tokenizerValue,
			)
			if err == nil {
				t.Fatal(
					"buildMemory() returned nil error",
				)
			}

			if !strings.Contains(
				err.Error(),
				tt.wantError,
			) {
				t.Fatalf(
					"buildMemory() error = %q, want it to contain %q",
					err.Error(),
					tt.wantError,
				)
			}
		})
	}
}

func TestBuildSlidingAgent(t *testing.T) {
	model := &stubModel{
		response: llm.Response{
			Message: llm.AssistantMessage(
				"factory agent works",
			),
			FinishReason: "stop",
			Usage: llm.Usage{
				InputTokens:  3,
				OutputTokens: 2,
				TotalTokens:  5,
			},
		},
	}

	builtAgent, err := Build(
		Config{
			MaxSteps: 4,
			Memory: MemoryConfig{
				Type:        MemoryTypeSliding,
				MaxMessages: 10,
			},
		},
		Dependencies{
			Model:    model,
			Registry: tool.NewRegistry(),
		},
	)
	if err != nil {
		t.Fatalf(
			"Build() returned error: %v",
			err,
		)
	}

	if builtAgent == nil {
		t.Fatal("Build() returned nil Agent")
	}

	result, err := builtAgent.Run(
		context.Background(),
		[]llm.Message{
			llm.UserMessage("hello"),
		},
	)
	if err != nil {
		t.Fatalf(
			"Agent.Run() returned error: %v",
			err,
		)
	}

	if got, want := result.FinalMessage.Content,
		"factory agent works"; got != want {
		t.Fatalf(
			"FinalMessage.Content = %q, want %q",
			got,
			want,
		)
	}

	if got, want := result.Steps, 1; got != want {
		t.Fatalf(
			"Steps = %d, want %d",
			got,
			want,
		)
	}
}

func TestBuildErrors(t *testing.T) {
	validModel := &stubModel{
		response: llm.Response{
			Message: llm.AssistantMessage("ok"),
		},
	}

	validConfig := Config{
		MaxSteps: 4,
		Memory: MemoryConfig{
			Type:        MemoryTypeSliding,
			MaxMessages: 10,
		},
	}

	tests := []struct {
		name      string
		cfg       Config
		deps      Dependencies
		wantError string
	}{
		{
			name: "invalid max steps",
			cfg: Config{
				MaxSteps: 0,
				Memory: MemoryConfig{
					Type:        MemoryTypeSliding,
					MaxMessages: 10,
				},
			},
			deps: Dependencies{
				Model:    validModel,
				Registry: tool.NewRegistry(),
			},
			wantError: "validate factory config",
		},
		{
			name: "model is missing",
			cfg:  validConfig,
			deps: Dependencies{
				Registry: tool.NewRegistry(),
			},
			wantError: "model is required",
		},
		{
			name: "registry is missing",
			cfg:  validConfig,
			deps: Dependencies{
				Model: validModel,
			},
			wantError: "tool registry is required",
		},
		{
			name: "tokenizer is missing",
			cfg: Config{
				MaxSteps: 4,
				Memory: MemoryConfig{
					Type:        MemoryTypeTokenBudget,
					TokenBudget: 100,
				},
			},
			deps: Dependencies{
				Model:    validModel,
				Registry: tool.NewRegistry(),
			},
			wantError: "tokenizer is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builtAgent, err := Build(
				tt.cfg,
				tt.deps,
			)
			if err == nil {
				t.Fatal("Build() returned nil error")
			}

			if builtAgent != nil {
				t.Fatal(
					"Build() returned non-nil Agent on error",
				)
			}

			if !strings.Contains(
				err.Error(),
				tt.wantError,
			) {
				t.Fatalf(
					"Build() error = %q, want it to contain %q",
					err.Error(),
					tt.wantError,
				)
			}
		})
	}
}
