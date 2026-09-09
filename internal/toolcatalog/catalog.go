package toolcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

type Source string

const (
	SourceLocal Source = "local"
	SourceMCP   Source = "mcp"
)

type Entry struct {
	Tool    tool.BaseTool
	Source  Source
	Enabled bool
}

type Item struct {
	Name        string
	Source      Source
	Enabled     bool
	Description string
	Parameters  string
}

type Catalog struct {
	entries []Entry
}

func New(entries ...Entry) (*Catalog, error) {
	copied := make([]Entry, len(entries))

	for i, entry := range entries {
		if entry.Tool == nil {
			return nil, fmt.Errorf("tool entry %d has no tool", i)
		}
		if entry.Source != SourceLocal && entry.Source != SourceMCP {
			return nil, fmt.Errorf(
				"tool entry %d has unsupported source %q",
				i,
				entry.Source,
			)
		}

		copied[i] = entry
	}

	return &Catalog{entries: copied}, nil
}

func (c *Catalog) EnabledTools() []tool.BaseTool {
	enabled := make([]tool.BaseTool, 0, len(c.entries))

	for _, entry := range c.entries {
		if entry.Enabled {
			enabled = append(enabled, entry.Tool)
		}
	}

	return enabled
}

func (c *Catalog) List(ctx context.Context) ([]Item, error) {
	items := make([]Item, 0, len(c.entries))

	for _, entry := range c.entries {
		info, err := entry.Tool.Info(ctx)
		if err != nil {
			return nil, fmt.Errorf("read tool info: %w", err)
		}
		if info == nil {
			return nil, fmt.Errorf("tool returned nil info")
		}
		if strings.TrimSpace(info.Name) == "" {
			return nil, fmt.Errorf("tool returned an empty name")
		}

		parameters := "{}"
		if info.ParamsOneOf != nil {
			parameterSchema, err := info.ParamsOneOf.ToJSONSchema()
			if err != nil {
				return nil, fmt.Errorf(
					"convert parameters for tool %q: %w",
					info.Name,
					err,
				)
			}

			data, err := json.Marshal(parameterSchema)
			if err != nil {
				return nil, fmt.Errorf(
					"encode parameters for tool %q: %w",
					info.Name,
					err,
				)
			}

			parameters = string(data)
		}

		items = append(items, Item{
			Name:        info.Name,
			Source:      entry.Source,
			Enabled:     entry.Enabled,
			Description: info.Desc,
			Parameters:  parameters,
		})
	}

	return items, nil
}
