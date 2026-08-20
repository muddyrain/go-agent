package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // user 消息携带
	ToolCallID string     `json:"tool_call_id,omitempty"` // role 为 "tool" 的消息携带
}

type Client struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func NewClient(baseURL string, apiKey string,
	model string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) Chat(
	ctx context.Context,
	messages []Message,
	tools []Tool,
) (Message, error) {
	payload := chatRequest{
		Model:    c.Model,
		Messages: messages,
		Tools:    tools,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Message{}, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return Message{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("http do: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Message{}, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return Message{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return Message{}, fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message, nil
}
