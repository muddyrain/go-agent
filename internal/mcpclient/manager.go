package mcpclient

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"agenthub/internal/tool"
)

const toolNamespaceSeparator = "__"

// Manager 管理 MCP Session，并把远程工具注册到 AgentHub Registry。
type Manager struct {
	mu       sync.Mutex
	registry *tool.Registry
	sessions map[string]Session
}

func NewManager(
	registry *tool.Registry,
) (*Manager, error) {
	if registry == nil {
		return nil, fmt.Errorf(
			"tool registry is required",
		)
	}

	return &Manager{
		registry: registry,
		sessions: make(map[string]Session),
	}, nil
}

func namespacedToolName(
	serverName string,
	remoteToolName string,
) string {
	return serverName +
		toolNamespaceSeparator +
		remoteToolName
}

func (m *Manager) AddServer(
	ctx context.Context,
	serverName string,
	session Session,
) error {
	serverName = strings.TrimSpace(
		serverName,
	)

	if serverName == "" {
		return fmt.Errorf(
			"MCP server name is required",
		)
	}

	if session == nil {
		return fmt.Errorf(
			"MCP session is required",
		)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 检查 Server 名称
	if _, exists := m.sessions[serverName]; exists {
		return fmt.Errorf(
			"MCP server %q is already registered",
			serverName,
		)
	}
	// 2. 发现远程工具
	remoteTools, err := session.ListTools(ctx)
	if err != nil {
		return fmt.Errorf(
			"list MCP server %q tools: %w",
			serverName,
			err,
		)
	}
	if len(remoteTools) == 0 {
		return fmt.Errorf(
			"MCP server %q exposed no tools",
			serverName,
		)
	}
	// 3. 创建 Adapter
	adapters := make(
		[]tool.Tool,
		0,
		len(remoteTools),
	)

	for _, remote := range remoteTools {
		remoteName := strings.TrimSpace(
			remote.Name,
		)

		localName := namespacedToolName(
			serverName,
			remoteName,
		)

		adapter, err := NewToolAdapter(
			session,
			localName,
			remote,
		)
		if err != nil {
			return fmt.Errorf(
				"adapt MCP server %q tool %q: %w",
				serverName,
				remote.Name,
				err,
			)
		}

		adapters = append(
			adapters,
			adapter,
		)
	}
	// 4. 原子注册
	if err := m.registry.RegisterBatch(
		adapters...,
	); err != nil {
		return fmt.Errorf(
			"register MCP server %q tools: %w",
			serverName,
			err,
		)
	}
	// 5. 记录 Session
	m.sessions[serverName] = session

	return nil
}
