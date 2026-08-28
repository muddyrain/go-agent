package tool

import (
	"context"
	"encoding/json"
	"testing"
)

func newTestFunction(t *testing.T, name string) *Function {
	t.Helper()

	function, err := NewFunction(
		Definition{
			Name:        name,
			Description: "test tool",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
		func(
			_ context.Context,
			_ json.RawMessage,
		) (string, error) {
			return name, nil
		},
	)
	if err != nil {
		t.Fatalf("NewFunction() returned error: %v", err)
	}

	return function
}

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewRegistry()
	echo := newTestFunction(t, "echo")

	if err := registry.Register(echo); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	got, ok := registry.Get("echo")
	if !ok {
		t.Fatal("Get() did not find registered tool")
	}

	if got != echo {
		t.Fatal("Get() did not return original tool")
	}

	gotWithSpaces, ok := registry.Get("  echo  ")
	if !ok {
		t.Fatal("Get() did not trim tool name")
	}

	if gotWithSpaces != echo {
		t.Fatal("Get() with spaces did not return original tool")
	}

	missing, ok := registry.Get("missing")
	if ok {
		t.Fatal("Get() found missing tool")
	}

	if missing != nil {
		t.Fatalf("Get() missing tool = %#v, want nil", missing)
	}
}

func TestRegistryRegisterValidation(t *testing.T) {
	t.Run("nil tool", func(t *testing.T) {
		registry := NewRegistry()

		err := registry.Register(nil)
		if err == nil {
			t.Fatal("Register() returned nil error, want error")
		}

		if got, want := err.Error(), "tool is required"; got != want {
			t.Fatalf("Register() error = %q, want %q", got, want)
		}
	})

	t.Run("duplicate name", func(t *testing.T) {
		registry := NewRegistry()

		first := newTestFunction(t, "echo")
		second := newTestFunction(t, "echo")

		if err := registry.Register(first); err != nil {
			t.Fatalf("first Register() returned error: %v", err)
		}

		err := registry.Register(second)
		if err == nil {
			t.Fatal("second Register() returned nil error, want error")
		}

		if got, want := err.Error(), `tool "echo" is already registered`; got != want {
			t.Fatalf("Register() error = %q, want %q", got, want)
		}
	})
}

func TestRegistryDefinitionsSorted(t *testing.T) {
	registry := NewRegistry()

	for _, name := range []string{
		"weather",
		"calculator",
		"echo",
	} {
		if err := registry.Register(newTestFunction(t, name)); err != nil {
			t.Fatalf("Register(%q) returned error: %v", name, err)
		}
	}

	definitions := registry.Definitions()

	if got, want := len(definitions), 3; got != want {
		t.Fatalf("Definitions() length = %d, want %d", got, want)
	}

	wantNames := []string{
		"calculator",
		"echo",
		"weather",
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
}

func TestRegistryDefinitionsReturnsCopy(t *testing.T) {
	registry := NewRegistry()
	echo := newTestFunction(t, "echo")

	if err := registry.Register(echo); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	first := registry.Definitions()
	if len(first) != 1 {
		t.Fatalf("first Definitions() length = %d, want 1", len(first))
	}

	first[0].Parameters[0] = 'X'

	second := registry.Definitions()
	if len(second) != 1 {
		t.Fatalf("second Definitions() length = %d, want 1", len(second))
	}

	if !json.Valid(second[0].Parameters) {
		t.Fatalf(
			"stored parameters were modified: %s",
			second[0].Parameters,
		)
	}
}

func TestRegistryValidate(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(
		newTestFunction(t, "echo"),
	); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	tests := []struct {
		name      string
		arguments json.RawMessage
		wantError bool
	}{
		{
			name:      "valid arguments",
			arguments: json.RawMessage(`{}`),
			wantError: false,
		},
		{
			name:      "wrong root type",
			arguments: json.RawMessage(`[]`),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Validate(
				"echo",
				tt.arguments,
			)

			if tt.wantError && err == nil {
				t.Fatal(
					"Validate() returned nil error, want error",
				)
			}

			if !tt.wantError && err != nil {
				t.Fatalf(
					"Validate() returned error: %v",
					err,
				)
			}
		})
	}
}

func TestRegistryRejectsInvalidSchema(t *testing.T) {
	function, err := NewFunction(
		Definition{
			Name:        "invalid",
			Description: "invalid schema tool",
			Parameters: json.RawMessage(
				`{"type":"not-a-real-json-type"}`,
			),
		},
		func(
			_ context.Context,
			_ json.RawMessage,
		) (string, error) {
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf(
			"NewFunction() returned error: %v",
			err,
		)
	}

	registry := NewRegistry()

	err = registry.Register(function)
	if err == nil {
		t.Fatal(
			"Register() returned nil error, want error",
		)
	}

	if _, ok := registry.Get("invalid"); ok {
		t.Fatal(
			"invalid schema tool was registered",
		)
	}
}
