package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"agenthub/examples/selfbuilt-runtime/internal/tool"
)

func TestManagerAddServersWithSameRemoteToolName(
	t *testing.T,
) {
	registry := tool.NewRegistry()

	manager, err := NewManager(registry)
	if err != nil {
		t.Fatalf(
			"NewManager() returned error: %v",
			err,
		)
	}

	filesystem := &stubSession{
		tools: []ToolDefinition{
			{
				Name:        "search",
				Description: "searches files",
				InputSchema: json.RawMessage(
					`{"type":"object"}`,
				),
			},
		},
		callResult: CallResult{
			Content: []Content{
				{
					Type: ContentTypeText,
					Text: "file result",
				},
			},
		},
	}

	database := &stubSession{
		tools: []ToolDefinition{
			{
				Name:        "search",
				Description: "searches rows",
				InputSchema: json.RawMessage(
					`{"type":"object"}`,
				),
			},
		},
		callResult: CallResult{
			Content: []Content{
				{
					Type: ContentTypeText,
					Text: "database result",
				},
			},
		},
	}

	if err := manager.AddServer(
		context.Background(),
		"filesystem",
		filesystem,
	); err != nil {
		t.Fatalf(
			"AddServer(filesystem) returned error: %v",
			err,
		)
	}

	if err := manager.AddServer(
		context.Background(),
		"database",
		database,
	); err != nil {
		t.Fatalf(
			"AddServer(database) returned error: %v",
			err,
		)
	}

	definitions := registry.Definitions()
	if got, want := len(definitions), 2; got != want {
		t.Fatalf(
			"Definitions() length = %d, want %d",
			got,
			want,
		)
	}

	wantNames := []string{
		"database__search",
		"filesystem__search",
	}

	for index, wantName := range wantNames {
		if got := definitions[index].Name; got != wantName {
			t.Fatalf(
				"Definitions()[%d].Name = %q, want %q",
				index,
				got,
				wantName,
			)
		}
	}

	executor, err := tool.NewExecutor(registry)
	if err != nil {
		t.Fatalf(
			"NewExecutor() returned error: %v",
			err,
		)
	}

	fileResult, err := executor.Execute(
		context.Background(),
		tool.Call{
			ID:        "call-filesystem",
			Name:      "filesystem__search",
			Arguments: json.RawMessage(`{}`),
		},
	)
	if err != nil {
		t.Fatalf(
			"filesystem Execute() returned error: %v",
			err,
		)
	}

	if got, want := fileResult.Content,
		"file result"; got != want {
		t.Fatalf(
			"filesystem result = %q, want %q",
			got,
			want,
		)
	}

	if got, want := filesystem.calledName,
		"search"; got != want {
		t.Fatalf(
			"filesystem remote name = %q, want %q",
			got,
			want,
		)
	}

	databaseResult, err := executor.Execute(
		context.Background(),
		tool.Call{
			ID:        "call-database",
			Name:      "database__search",
			Arguments: json.RawMessage(`{}`),
		},
	)
	if err != nil {
		t.Fatalf(
			"database Execute() returned error: %v",
			err,
		)
	}

	if got, want := databaseResult.Content,
		"database result"; got != want {
		t.Fatalf(
			"database result = %q, want %q",
			got,
			want,
		)
	}

	if got, want := database.calledName,
		"search"; got != want {
		t.Fatalf(
			"database remote name = %q, want %q",
			got,
			want,
		)
	}

}

func TestNewManagerValidation(t *testing.T) {
	manager, err := NewManager(nil)

	if err == nil {
		t.Fatal(
			"NewManager() returned nil error",
		)
	}

	if manager != nil {
		t.Fatal(
			"NewManager() returned non-nil manager",
		)
	}
}

func TestManagerAddServerValidation(t *testing.T) {
	registry := tool.NewRegistry()

	manager, err := NewManager(registry)
	if err != nil {
		t.Fatalf(
			"NewManager() returned error: %v",
			err,
		)
	}

	tests := []struct {
		name       string
		serverName string
		session    Session
		wantError  string
	}{
		{
			name:       "empty server name",
			serverName: " ",
			session:    &stubSession{},
			wantError:  "server name is required",
		},
		{
			name:       "nil session",
			serverName: "filesystem",
			session:    nil,
			wantError:  "session is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.AddServer(
				context.Background(),
				tt.serverName,
				tt.session,
			)

			if err == nil {
				t.Fatal(
					"AddServer() returned nil error",
				)
			}

			if !strings.Contains(
				err.Error(),
				tt.wantError,
			) {
				t.Fatalf(
					"error = %q, want it to contain %q",
					err.Error(),
					tt.wantError,
				)
			}
		})
	}
}

func TestManagerRejectsDuplicateServer(
	t *testing.T,
) {
	registry := tool.NewRegistry()

	manager, err := NewManager(registry)
	if err != nil {
		t.Fatalf(
			"NewManager() returned error: %v",
			err,
		)
	}

	first := &stubSession{
		tools: []ToolDefinition{
			{
				Name:        "search",
				Description: "searches files",
				InputSchema: json.RawMessage(
					`{"type":"object"}`,
				),
			},
		},
	}

	if err := manager.AddServer(
		context.Background(),
		"filesystem",
		first,
	); err != nil {
		t.Fatalf(
			"first AddServer() returned error: %v",
			err,
		)
	}

	second := &stubSession{
		tools: []ToolDefinition{
			{
				Name:        "read",
				Description: "reads files",
				InputSchema: json.RawMessage(
					`{"type":"object"}`,
				),
			},
		},
	}

	err = manager.AddServer(
		context.Background(),
		"filesystem",
		second,
	)
	if err == nil {
		t.Fatal(
			"second AddServer() returned nil error",
		)
	}

	if _, ok := registry.Get(
		"filesystem__read",
	); ok {
		t.Fatal(
			"duplicate server registered second session tools",
		)
	}
}

func TestManagerAddServerCanRetryAfterRegistrationFailure(
	t *testing.T,
) {
	registry := tool.NewRegistry()

	manager, err := NewManager(registry)
	if err != nil {
		t.Fatalf(
			"NewManager() returned error: %v",
			err,
		)
	}

	invalid := &stubSession{
		tools: []ToolDefinition{
			{
				Name:        "valid",
				Description: "valid tool",
				InputSchema: json.RawMessage(
					`{"type":"object"}`,
				),
			},
			{
				Name:        "invalid",
				Description: "invalid tool",
				InputSchema: json.RawMessage(
					`{"type":"not-a-real-json-type"}`,
				),
			},
		},
	}

	err = manager.AddServer(
		context.Background(),
		"filesystem",
		invalid,
	)
	if err == nil {
		t.Fatal(
			"AddServer() returned nil error",
		)
	}

	if _, ok := registry.Get(
		"filesystem__valid",
	); ok {
		t.Fatal(
			"failed server partially registered valid tool",
		)
	}

	valid := &stubSession{
		tools: []ToolDefinition{
			{
				Name:        "search",
				Description: "searches files",
				InputSchema: json.RawMessage(
					`{"type":"object"}`,
				),
			},
		},
	}

	if err := manager.AddServer(
		context.Background(),
		"filesystem",
		valid,
	); err != nil {
		t.Fatalf(
			"retry AddServer() returned error: %v",
			err,
		)
	}

	if _, ok := registry.Get(
		"filesystem__search",
	); !ok {
		t.Fatal(
			"retry did not register valid tool",
		)
	}
}

func TestManagerAddServerListToolsError(
	t *testing.T,
) {
	registry := tool.NewRegistry()

	manager, err := NewManager(registry)
	if err != nil {
		t.Fatalf(
			"NewManager() returned error: %v",
			err,
		)
	}

	listErr := errors.New(
		"remote list failed",
	)

	session := &stubSession{
		err: listErr,
	}

	err = manager.AddServer(
		context.Background(),
		"filesystem",
		session,
	)

	if !errors.Is(err, listErr) {
		t.Fatalf(
			"AddServer() error = %v, want wrapped list error",
			err,
		)
	}
}
