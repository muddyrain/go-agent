package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agenthub/internal/tool"
)

// ToolAdapter 把一个 MCP 远程工具适配成 AgentHub 的 tool.Tool。
type ToolAdapter struct {
	session    Session
	remoteName string
	definition tool.Definition
}

func NewToolAdapter(
	session Session,
	remote ToolDefinition,
) (*ToolAdapter, error) {
	if session == nil {
		return nil, fmt.Errorf(
			"MCP session is required",
		)
	}

	name := strings.TrimSpace(remote.Name)
	if name == "" {
		return nil, fmt.Errorf(
			"MCP tool name is required",
		)
	}

	if len(remote.InputSchema) == 0 {
		return nil, fmt.Errorf(
			"MCP tool %q input schema is required",
			name,
		)
	}

	if !json.Valid(remote.InputSchema) {
		return nil, fmt.Errorf(
			"MCP tool %q input schema must be valid JSON",
			name,
		)
	}

	schema := cloneJSON(remote.InputSchema)

	return &ToolAdapter{
		session:    session,
		remoteName: name,
		definition: tool.Definition{
			Name: name,
			Description: strings.TrimSpace(
				remote.Description,
			),
			Parameters: schema,
		},
	}, nil
}

func cloneJSON(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}

	cloned := make(
		json.RawMessage,
		len(value),
	)
	copy(cloned, value)

	return cloned
}

func (t *ToolAdapter) Definition() tool.Definition {
	definition := t.definition
	definition.Parameters = cloneJSON(
		definition.Parameters,
	)

	return definition
}

func (t *ToolAdapter) Execute(
	ctx context.Context,
	arguments json.RawMessage,
) (string, error) {
	result, err := t.session.CallTool(
		ctx,
		t.remoteName,
		cloneJSON(arguments),
	)
	if err != nil {
		return "", fmt.Errorf(
			"call MCP tool %q: %w",
			t.remoteName,
			err,
		)
	}

	content, err := formatContent(
		result.Content,
	)
	if err != nil {
		return "", fmt.Errorf(
			"format MCP tool %q result: %w",
			t.remoteName,
			err,
		)
	}

	if result.IsError {
		if content == "" {
			return "", fmt.Errorf(
				"MCP tool %q failed",
				t.remoteName,
			)
		}

		return "", fmt.Errorf(
			"MCP tool %q failed: %s",
			t.remoteName,
			content,
		)
	}

	return content, nil
}

func formatContent(
	contents []Content,
) (string, error) {
	parts := make(
		[]string,
		0,
		len(contents),
	)

	for _, content := range contents {
		switch content.Type {
		case ContentTypeText:
			parts = append(
				parts,
				content.Text,
			)

		default:
			return "", fmt.Errorf(
				"unsupported content type %q",
				content.Type,
			)
		}
	}

	return strings.Join(parts, "\n"), nil
}

var _ tool.Tool = (*ToolAdapter)(nil)
