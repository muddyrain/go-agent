package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

// ServerConfig 描述应用本次要启动的一份 MCP Server 配置。
// 当前一份配置创建一个 Session；Name 是 AgentHub 本地标识，Command 和
// Args 决定实际启动哪个 stdio 子进程。
type ServerConfig struct {
	Name    string
	Command string
	Args    []string
}

// DiscoveredTool 把模型可调用的命名空间 Tool 与来源 Server 关联起来，
// 供入口继续登记进统一 Tool Catalog。
type DiscoveredTool struct {
	Tool   tool.BaseTool
	Server string
}

// managedSession 保存关闭连接时需要的 Session 及可读 Server 名称。
// 它不代表“同一 Server 的多个 Client”，而是 Manager 内的一份连接记录。
type managedSession struct {
	name    string
	session *Session
}

// Manager 聚合多个 MCP Server 的 Session 和已发现工具。
// 它负责批量建立连接、应用模型侧命名空间以及统一释放资源；每条连接
// 的握手、工具发现和关闭细节仍由 Session 负责。
type Manager struct {
	sessions []managedSession
	tools    []DiscoveredTool
}

// OpenServers 按配置顺序建立多个 MCP Session，并把每个远程工具包装成
// server__tool 的模型侧唯一名称。若任一步失败，会关闭此前已经成功建立
// 的 Session，避免返回半初始化 Manager 或遗留子进程。
func OpenServers(
	ctx context.Context,
	configs []ServerConfig,
) (*Manager, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf(
			"at least one MCP server config is required",
		)
	}

	manager := &Manager{}
	// 空 struct 不占额外业务数据，适合把 map 当作集合使用；这里只需
	// 判断 Server 名是否出现过，不需要为名称保存额外值。
	serverNames := make(
		map[string]struct{},
		len(configs),
	)

	for index, config := range configs {
		config.Name = strings.TrimSpace(config.Name)
		if config.Name == "" {
			return nil, errors.Join(
				fmt.Errorf(
					"MCP server config %d has an empty name",
					index,
				),
				manager.Close(),
			)
		}

		if _, exists := serverNames[config.Name]; exists {
			return nil, errors.Join(
				fmt.Errorf(
					"MCP server %q is configured more than once",
					config.Name,
				),
				manager.Close(),
			)
		}
		serverNames[config.Name] = struct{}{}

		mcpSession, err := OpenStdio(
			ctx,
			config.Command,
			config.Args...,
		)
		if err != nil {
			return nil, errors.Join(
				fmt.Errorf(
					"open MCP server %q: %w",
					config.Name,
					err,
				),
				manager.Close(),
			)
		}

		// Session 建立成功后立即登记进 Manager。这样后续工具适配失败时，
		// manager.Close 也能回收刚启动的这一条连接，而不只关闭更早的连接。
		manager.sessions = append(
			manager.sessions,
			managedSession{
				name:    config.Name,
				session: mcpSession,
			},
		)

		for _, remoteTool := range mcpSession.Tools() {
			// Session 返回的是 MCP Server 原始工具；Manager 再包一层模型侧
			// 命名空间，解决不同 Server 暴露同名工具时的 ToolsNode 冲突。
			namespacedTool, err := newNamespacedTool(
				ctx,
				config.Name,
				remoteTool,
			)
			if err != nil {
				return nil, errors.Join(
					fmt.Errorf(
						"adapt MCP server %q tool: %w",
						config.Name,
						err,
					),
					manager.Close(),
				)
			}

			manager.tools = append(
				manager.tools,
				DiscoveredTool{
					Tool:   namespacedTool,
					Server: config.Name,
				},
			)
		}
	}

	return manager, nil
}

// Tools 返回新的切片结构，避免入口在组装 Catalog 时修改 Manager
// 内部的发现结果；其中 Tool 实例仍是相同的命名空间代理。
func (m *Manager) Tools() []DiscoveredTool {
	if m == nil {
		return nil
	}

	return append(
		[]DiscoveredTool(nil),
		m.tools...,
	)
}

// Close 按创建顺序的逆序关闭所有 Session，并聚合全部关闭错误。
// 倒序与常见资源栈语义一致：后建立的资源通常更依赖先建立的环境；
// 即使某一条关闭失败，也继续尝试释放剩余连接。
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}

	var closeErrors []error

	for index := len(m.sessions) - 1; index >= 0; index-- {
		managed := m.sessions[index]

		if err := managed.session.Close(); err != nil {
			closeErrors = append(
				closeErrors,
				fmt.Errorf(
					"close MCP server %q: %w",
					managed.name,
					err,
				),
			)
		}
	}

	// 清空引用既让重复 Close 不再遍历旧 Session，也表示 Manager 已不再
	// 对外提供这些工具；Session 自身的 Close 仍负责真正的幂等保护。
	m.sessions = nil
	m.tools = nil

	return errors.Join(closeErrors...)
}
