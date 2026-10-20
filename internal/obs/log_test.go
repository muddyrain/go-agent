package obs

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestInfoOutputsJSON(t *testing.T) {
	origOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = origOut }()

	logger := New()
	logger.Info("test message", String("env", "development"), Int("port", 8080))

	w.Close()
	outputBytes, _ := io.ReadAll(r)
	output := strings.TrimSpace(string(outputBytes))

	var m map[string]any
	if err := json.Unmarshal(outputBytes, &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if m["level"] != "info" {
		t.Errorf("level = %v, want info", m["level"])
	}
	if m["msg"] != "test message" {
		t.Errorf("msg = %v, want test message", m["msg"])
	}
	if m["env"] != "development" {
		t.Errorf("env = %v, want development", m["env"])
	}
	if m["port"] != float64(8080) {
		// JSON 数字反序列化成 float64
		t.Errorf("port = %v, want 8080", m["port"])
	}
	if _, ok := m["ts"]; !ok {
		t.Error("ts field is missing")
	}
}

func TestErrorOutputsJSON(t *testing.T) {
	origOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = origOut }()

	logger := New()
	logger.Error("something broke", String("node", "chat_model"))

	w.Close()
	outputBytes, _ := io.ReadAll(r)
	output := strings.TrimSpace(string(outputBytes))

	var m map[string]any
	if err := json.Unmarshal(outputBytes, &m); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	if m["level"] != "error" {
		t.Errorf("level = %v, want error", m["level"])
	}
	if m["node"] != "chat_model" {
		t.Errorf("node = %v, want chat_model", m["node"])
	}
}
