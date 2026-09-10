package mcpclient

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

type ServerConfig struct {
	Name    string
	Command string
	Args    []string
}

type DiscoveredTool struct {
	Tool   tool.BaseTool
	Server string
}

type managedSession struct {
	name    string
	session *Session
}

type Manager struct {
	sessions []managedSession
	tools    []DiscoveredTool
}

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

		manager.sessions = append(
			manager.sessions,
			managedSession{
				name:    config.Name,
				session: mcpSession,
			},
		)

		for _, remoteTool := range mcpSession.Tools() {
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

func (m *Manager) Tools() []DiscoveredTool {
	if m == nil {
		return nil
	}

	return append(
		[]DiscoveredTool(nil),
		m.tools...,
	)
}

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

	m.sessions = nil
	m.tools = nil

	return errors.Join(closeErrors...)
}
