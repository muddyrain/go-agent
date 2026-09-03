package agentfactory

import (
	"reflect"
	"testing"

	"agenthub/internal/config"
)

func TestConfigFromApp(t *testing.T) {
	input := config.AgentConfig{
		MaxSteps: 6,
		Memory: config.AgentMemoryConfig{
			Type:        "token_budget",
			MaxMessages: 30,
			TokenBudget: 2048,
		},
	}

	got := ConfigFromApp(input)

	want := Config{
		MaxSteps: 6,
		Memory: MemoryConfig{
			Type:        MemoryTypeTokenBudget,
			MaxMessages: 30,
			TokenBudget: 2048,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"ConfigFromApp() = %#v, want %#v",
			got,
			want,
		)
	}
}
