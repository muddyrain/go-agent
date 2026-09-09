package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeConfig, "invalid config")

	if err == nil {
		t.Fatal("New() returned nil")
	}

	if got, want := err.Error(), "invalid config"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	if got, want := CodeOf(err), CodeConfig; got != want {
		t.Fatalf("CodeOf() = %q, want %q", got, want)
	}

	var appErr *Error
	if !errors.As(err, &appErr) {
		t.Fatal("errors.As() could not extract *Error")
	}

	if appErr.Err != nil {
		t.Fatalf("underlying error = %v, want nil", appErr.Err)
	}
}

func TestWrap(t *testing.T) {
	rootErr := errors.New("file not found")

	err := Wrap(CodeConfig, "read config", rootErr)
	if err == nil {
		t.Fatal("Wrap() returned nil")
	}

	if got, want := err.Error(), "read config: file not found"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	if got, want := CodeOf(err), CodeConfig; got != want {
		t.Fatalf("CodeOf() = %q, want %q", got, want)
	}

	if !errors.Is(err, rootErr) {
		t.Fatal("errors.Is() could not find root error")
	}
}

func TestWrapNil(t *testing.T) {
	err := Wrap(CodeConfig, "read config", nil)

	if err != nil {
		t.Fatalf("Wrap() = %v, want nil", err)
	}
}

func TestCodeOf(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Code
	}{
		{
			name: "application error",
			err:  New(CodeConfig, "invalid config"),
			want: CodeConfig,
		},
		{
			name: "application error inside another wrapper",
			err:  fmt.Errorf("start application: %w", New(CodeConfig, "invalid config")),
			want: CodeConfig,
		},
		{
			name: "plain error",
			err:  errors.New("unknown error"),
			want: CodeInternal,
		},
		{
			name: "nil error",
			err:  nil,
			want: CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CodeOf(tt.err)
			if got != tt.want {
				t.Fatalf("CodeOf() = %q, want %q", got, tt.want)
			}
		})
	}
}
