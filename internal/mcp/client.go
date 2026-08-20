package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Client 是 MCP 客户端，通过 stdio 连接 MCP Server
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner

	mu      sync.Mutex
	nextID  int
	pending map[int]chan json.RawMessage // 等待响应的请求
}

// jsonRPCRequest 是 JSON-RPC 2.0 请求
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonRPCResponse 是 JSON-RPC 2.0 响应
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// jsonRPCNotification 是 JSON-RPC 2.0 通知（没有 id）
type jsonRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Start 启动 MCP Server 子进程并开始读取 stdout
func (c *Client) Start(ctx context.Context, command string, args ...string) error {
	c.cmd = exec.CommandContext(ctx, command, args...)
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	c.stdout = bufio.NewScanner(stdout)
	c.stdout.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB 缓冲

	c.pending = make(map[int]chan json.RawMessage)

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	// 启动 goroutine 持续读取 stdout
	go c.readLoop()
	return nil
}

// readLoop 持续读取 Server 的 stdout，分发响应
func (c *Client) readLoop() {
	for c.stdout.Scan() {
		line := c.stdout.Text()
		if line == "" {
			continue
		}
		var resp jsonRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue // 忽略无法解析的行（可能是通知）
		}

		// 找到对应的 pending channel
		c.mu.Lock()
		ch, ok := c.pending[resp.ID]
		delete(c.pending, resp.ID)
		c.mu.Unlock()

		if ok {
			if resp.Error != nil {
				ch <- nil // 简单处理，错误后续可扩展
			} else {
				ch <- resp.Result
			}
		}
	}
}

// sendRequest 发送一个 JSON-RPC 请求，等待响应
func (c *Client) sendRequest(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan json.RawMessage, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)

	if err != nil {
		return nil, err
	}
	data = append(data, '\n') // 每行一个消息

	if _, err := c.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	result := <-ch
	return result, nil
}

// Initialize 执行 MCP 初始化握手
func (c *Client) Initialize() error {
	// 1. 发送 initialize 请求
	_, err := c.sendRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "go-agent",
			"version": "0.1",
		},
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	// 2. 发送 initialized 通知
	notif := jsonRPCNotification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	data, _ := json.Marshal(notif)
	data = append(data, '\n')
	c.stdin.Write(data)

	return nil
}

// ToolInfo 是 MCP Server 返回的工具定义
type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ListTools 列出 MCP Server 提供的所有工具
func (c *Client) ListTools() ([]ToolInfo, error) {
	result, err := c.sendRequest("tools/list", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Tools []ToolInfo `json:"tools"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal tools: %w", err)
	}
	return resp.Tools, nil
}

// CallTool 调用 MCP Server 的工具
func (c *Client) CallTool(name string, arguments json.RawMessage) (string, error) {
	result, err := c.sendRequest("tools/call", map[string]interface{}{
		"name":      name,
		"arguments": json.RawMessage(arguments),
	})
	if err != nil {
		return "", err
	}

	// MCP tools/call 返回 content 数组，每个元素有 type 和 text
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}

	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("unmarshal tool result: %w", err)
	}

	if resp.IsError {
		return "", fmt.Errorf("tool execution error")
	}

	// 拼接所有 text 内容
	var output string
	for _, item := range resp.Content {
		if item.Type == "text" {
			output += item.Text
		}
	}
	return output, nil
}

// Close 关闭 MCP Client，终止子进程
func (c *Client) Close() error {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Wait()
	}
	return nil
}
