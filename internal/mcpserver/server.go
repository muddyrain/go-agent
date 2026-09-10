package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// 这两个值是 AgentHub 选择教学 MCP Server 配置时使用的逻辑 ID，
// 不是 MCP 协议中的工具名。两个 Server 都可以注册同名 add_numbers。
const (
	ServerCalculator = "calculator"
	ServerAccounting = "accounting"
)

// serverProfile 保存多个教学 Server 之间真正不同的配置。
// 协议启动和加法 Handler 共享同一份实现，避免为了两个 Server 复制流程。
type serverProfile struct {
	name            string
	toolDescription string
	resultLabel     string
}

var serverProfiles = map[string]serverProfile{
	ServerCalculator: {
		name:            "agenthub-calculator-mcp",
		toolDescription: "计算两个普通数字的和",
		resultLabel:     "calculator",
	},
	ServerAccounting: {
		name:            "agenthub-accounting-mcp",
		toolDescription: "计算两个账务数字的和",
		resultLabel:     "accounting",
	},
}

// ServeStdio 把当前进程作为指定 profile 的 MCP Server 运行。
// ServeStdio 接管 stdin/stdout 传输 JSON-RPC；运行期间不能向 stdout 打印
// 普通日志，否则日志文本会混入协议消息并破坏 Client 解码。
func ServeStdio(serverID string) error {

	profile, ok := serverProfiles[serverID]
	if !ok {
		return fmt.Errorf("unknown MCP server %q", serverID)
	}

	mcpServer := server.NewMCPServer(
		profile.name,
		"0.1.0",
	)

	// AddTool 同时注册工具 Schema 和执行 Handler：tools/list 返回前者，
	// tools/call 按原始名称 add_numbers 找到后者。模型侧命名空间只存在于
	// AgentHub Client 的代理层，不会改变这里的远程注册名。
	mcpServer.AddTool(
		mcp.NewTool(
			"add_numbers",
			mcp.WithDescription(profile.toolDescription),
			mcp.WithNumber(
				"a",
				mcp.Required(),
				mcp.Description("第一个数字"),
			),
			mcp.WithNumber(
				"b",
				mcp.Required(),
				mcp.Description("第二个数字"),
			),
		),
		newAddNumbersHandler(profile.resultLabel),
	)

	return server.ServeStdio(mcpServer)
}

// newAddNumbersHandler 返回捕获 resultLabel 的闭包，让多个 Server 复用
// 参数校验和计算逻辑，同时在结果中保留可观察的来源标签。
func newAddNumbersHandler(
	resultLabel string,
) server.ToolHandlerFunc {
	return func(
		_ context.Context,
		request mcp.CallToolRequest,
	) (*mcp.CallToolResult, error) {
		a, err := request.RequireFloat("a")
		if err != nil {
			return nil, fmt.Errorf("read argument a: %w", err)
		}

		b, err := request.RequireFloat("b")
		if err != nil {
			return nil, fmt.Errorf("read argument b: %w", err)
		}

		return mcp.NewToolResultText(
			fmt.Sprintf(
				"[%s] %g + %g = %g",
				resultLabel,
				a,
				b,
				a+b,
			),
		), nil
	}
}
