package main

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

func TestEinoToolLoop(t *testing.T) {
	ctx := context.Background()

	weatherTool, err := newWeatherTool()
	if err != nil {
		t.Fatalf("newWeatherTool() error = %v", err)
	}

	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: &demoChatModel{},
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{weatherTool},
		},
	})
	if err != nil {
		t.Fatalf("react.NewAgent() error = %v", err)
	}

	messageOption, messageFuture := react.WithMessageFuture()
	response, err := reactAgent.Generate(
		ctx,
		[]*schema.Message{schema.UserMessage("杭州今天天气怎么样？")},
		messageOption,
	)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var messages []*schema.Message
	iterator := messageFuture.GetMessages()
	for {
		message, ok, err := iterator.Next()
		if err != nil {
			t.Fatalf("read ReAct message: %v", err)
		}
		if !ok {
			break
		}
		messages = append(messages, message)
	}

	if got, want := len(messages), 3; got != want {
		t.Fatalf("len(messages) = %d, want %d", got, want)
	}

	toolCallMessage := messages[0]
	if got, want := toolCallMessage.Role, schema.Assistant; got != want {
		t.Fatalf("messages[0].Role = %q, want %q", got, want)
	}
	if got, want := len(toolCallMessage.ToolCalls), 1; got != want {
		t.Fatalf("len(messages[0].ToolCalls) = %d, want %d", got, want)
	}

	call := toolCallMessage.ToolCalls[0]
	if got, want := call.Function.Name, "get_weather"; got != want {
		t.Fatalf("ToolCall.Function.Name = %q, want %q", got, want)
	}

	toolMessage := messages[1]
	if got, want := toolMessage.Role, schema.Tool; got != want {
		t.Fatalf("messages[1].Role = %q, want %q", got, want)
	}
	if got, want := toolMessage.ToolName, call.Function.Name; got != want {
		t.Fatalf("ToolMessage.ToolName = %q, want %q", got, want)
	}
	if got, want := toolMessage.ToolCallID, call.ID; got != want {
		t.Fatalf("ToolMessage.ToolCallID = %q, want %q", got, want)
	}

	finalMessage := messages[2]
	if got, want := finalMessage.Role, schema.Assistant; got != want {
		t.Fatalf("messages[2].Role = %q, want %q", got, want)
	}
	if got, want := response.Content, finalMessage.Content; got != want {
		t.Fatalf("Generate() response = %q, want %q", got, want)
	}
	if !strings.Contains(finalMessage.Content, toolMessage.Content) {
		t.Fatalf(
			"final message %q does not contain tool result %q",
			finalMessage.Content,
			toolMessage.Content,
		)
	}
}

func TestDemoChatModelRequiresWeatherTool(t *testing.T) {
	_, err := (&demoChatModel{}).Generate(
		context.Background(),
		[]*schema.Message{schema.UserMessage("杭州今天天气怎么样？")},
	)
	if err == nil {
		t.Fatal("Generate() error = nil, want missing tool error")
	}
	if !strings.Contains(err.Error(), "no tools bound") {
		t.Fatalf("Generate() error = %q, want missing tool detail", err)
	}
}
