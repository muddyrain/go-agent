package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	mcpadapter "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// SessionState 表示 AgentHub 当前已知的 MCP 连接状态。
//
// 它不是主动健康检查结果，而是根据连接建立、调用错误和关闭操作
// 记录的本地状态。
type SessionState string

const (
	// SessionStateReady 表示初始化和工具发现成功，可以调用工具。
	SessionStateReady SessionState = "ready"
	// SessionStateUnavailable 表示已观察到连接层错误，
	// 当前 Session 不应继续接受新的工具调用。
	SessionStateUnavailable SessionState = "unavailable"
	// SessionStateClosed 表示 Session 已主动释放资源。
	SessionStateClosed SessionState = "closed"
)

// Session 表示 AgentHub 到一个 MCP Server 的单条真实连接。
// 它持有底层 MCP Client 和该连接动态发现的原始 Eino Tools；多 Server
// 聚合、模型侧命名空间及应用配置分别由 Manager 和入口负责。
type Session struct {
	client *client.Client
	tools  []tool.BaseTool

	// stateMu 保护 state，避免工具调用和关闭操作并发读写状态。
	stateMu sync.RWMutex
	state   SessionState

	// closeOnce 保证底层 MCP Client 最多只关闭一次。
	// closeErr 保存第一次关闭的结果，供后续 Close 调用返回。
	closeOnce sync.Once
	closeErr  error
}

// OpenStdio 启动一个 stdio MCP Server 子进程，并依次完成：
// 创建 Client/管道 → Initialize 协议握手 → tools/list 工具发现 →
// Eino MCP Adapter 转换。只有整条链成功后才返回 ready Session。
func OpenStdio(
	ctx context.Context,
	command string,
	args ...string,
) (*Session, error) {
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("MCP server command is empty")
	}

	mcpClient, err := client.NewStdioMCPClientWithOptions(
		command,
		nil,
		args,
		transport.WithCommandFunc(
			func(
				startCtx context.Context,
				command string,
				_ []string,
				args []string,
			) (*exec.Cmd, error) {
				// MCP SDK 仍负责管道、JSON-RPC 与进程等待；这里只定制
				// 子进程创建方式，以控制它继承哪些环境变量。
				cmd := exec.CommandContext(startCtx, command, args...)

				// 不把 AgentHub 进程中的模型 API Key 等环境变量
				// 默认传递给 MCP Server 子进程。
				cmd.Env = []string{}

				return cmd, nil
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start MCP stdio client: %w", err)
	}

	// Initialize 协商 MCP 协议版本和双方能力；握手前不能调用 tools/list
	// 或 tools/call。这里声明的是 AgentHub 作为 MCP Client 的身份。
	initializeRequest := mcp.InitializeRequest{}
	initializeRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initializeRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "agenthub",
		Version: "0.1.0",
	}

	// 初始化失败后仍需关闭已启动的 stdio Client；
	// errors.Join 同时保留握手错误和可能出现的清理错误。
	if _, err := mcpClient.Initialize(ctx, initializeRequest); err != nil {
		return nil, errors.Join(
			fmt.Errorf("initialize MCP session: %w", err),
			mcpClient.Close(),
		)
	}

	// GetTools 内部先调用 MCP tools/list，再把每个远程 Tool 定义包装成
	// Eino tool.BaseTool。包装后的 InvokableRun 会通过同一 Client 发起
	// tools/call；Session 本身不执行工具 Handler。
	tools, err := mcpadapter.GetTools(
		ctx,
		&mcpadapter.Config{
			Cli: mcpClient,
		},
	)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("discover MCP tools: %w", err),
			mcpClient.Close(),
		)
	}

	return &Session{
		client: mcpClient,
		tools:  tools,
		state:  SessionStateReady,
	}, nil
}

// Tools 返回新的切片结构，避免调用者 append、排序或覆盖元素时修改
// Session 内部的工具列表；其中的 Tool 实例仍是同一批 Adapter 对象。
func (s *Session) Tools() []tool.BaseTool {
	return append([]tool.BaseTool(nil), s.tools...)
}

// State 返回 AgentHub 当前记录的 Session 状态。
func (s *Session) State() SessionState {
	if s == nil {
		return SessionStateClosed
	}

	s.stateMu.RLock()
	defer s.stateMu.RUnlock()

	return s.state
}

// ensureReady 在工具调用前检查当前 Session 是否还能使用。
func (s *Session) ensureReady() error {
	switch state := s.State(); state {
	case SessionStateReady:
		return nil
	case SessionStateUnavailable:
		return fmt.Errorf("MCP session is unavailable")
	case SessionStateClosed:
		return fmt.Errorf("MCP session is closed")
	default:
		return fmt.Errorf("MCP session has invalid state %q", state)
	}
}

// markUnavailable 记录已经观察到的连接层错误。
// closed 是终止状态，不能再退回 unavailable。
func (s *Session) markUnavailable() {
	if s == nil {
		return
	}

	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.state != SessionStateClosed {
		s.state = SessionStateUnavailable
	}
}

// Close 释放 MCP Client、stdio 管道和 Server 子进程。
// 该方法是幂等的：重复调用不会重复执行底层关闭操作。
func (s *Session) Close() error {
	if s == nil {
		return nil
	}

	s.closeOnce.Do(func() {
		s.stateMu.Lock()
		s.state = SessionStateClosed
		s.stateMu.Unlock()

		if s.client != nil {
			s.closeErr = s.client.Close()
		}
	})

	return s.closeErr
}
