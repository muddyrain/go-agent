package llm

import (
	"bufio"
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
	MaxRetries int // 新增：最大重试次数
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

// 流式请求体
type chatStreamRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
	Stream   bool      `json:"stream"`
}

// 流式响应体（注意是 delta，不是 message）
type chatStreamResponse struct {
	Choices []struct {
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// StreamChunk 是流式返回的一个数据块
type StreamChunk struct {
	Content string // 增量文本内容
	Err     error  // 流过程中出错时携带
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
		MaxRetries: 3, // 默认重试 3 次
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

	var resp *http.Response

	// 重试循环：第 0 次是正常请求，第 1~MaxRetries 次是重试

	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		// 每次重试都重新创建 request（因为 body 读一次就没了）
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
		if reqErr != nil {
			return Message{}, fmt.Errorf("create request: %w", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		resp, err = c.HTTPClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break // 成功，或 4xx（客户端错误，不重试）
		}
		// 到这里说明需要重试
		if resp != nil {
			resp.Body.Close() // 重试前必须关闭上一次的响应体
		}

		if attempt < c.MaxRetries {
			// 指数退避：第 1 次等 1s，第 2 次等 2s，第 3 次等 4s
			wait := time.Duration(1<<attempt) * time.Second

			if err != nil {
				fmt.Printf("请求失败（%v），%v 后第 %d 次重试...\n", err, wait, attempt+1)
			} else {
				fmt.Printf("请求失败（status %d），%v 后第 %d 次重试...\n", resp.StatusCode, wait, attempt+1)
			}
			time.Sleep(wait)
		}
	}

	// 循环结束后，检查最终结果
	if err != nil {
		return Message{}, fmt.Errorf("http do (after retries): %w", err)
	}

	defer resp.Body.Close()

	// 读 body、判状态码、Unmarshal
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

func (c *Client) ChatStream(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error) {
	// 1. 构造流式请求（Stream: true）

	payload := chatStreamRequest{
		Model:    c.Model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	// 2. 发请求
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}

	// 3. 非 2xx 直接报错（流式也可能返回 401/404 等）
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	// 4. 创建 channel，启动 goroutine 读 SSE 流
	ch := make(chan StreamChunk)

	go func() {
		defer close(ch)         // goroutine 结束时关闭 channel
		defer resp.Body.Close() // 关闭响应体

		scanner := bufio.NewScanner(resp.Body)

		for scanner.Scan() {
			line := scanner.Text()
			// SSE 行格式："data: {...}"，空行跳过
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			// 流结束标志
			if data == "[DONE]" {
				return
			}
			// 解析这一个 chunk 的 JSON
			var streamResp chatStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				ch <- StreamChunk{Err: fmt.Errorf("unmarshal chunk: %w", err)}
				return
			}
			if len(streamResp.Choices) == 0 {
				continue
			}

			// 把增量内容发送到 channel
			content := streamResp.Choices[0].Delta.Content

			if content != "" {
				ch <- StreamChunk{Content: content}
			}
		}

		// scanner 出错
		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("scan stream: %w", err)}
		}
	}()

	return ch, nil
}
