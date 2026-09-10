package toolcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

// Source 标识工具由 AgentHub 本地实现还是由 MCP Server 动态发现。
// 它只用于目录分类和诊断，不参与 ToolCall 路由。
type Source string

const (
	SourceLocal Source = "local"
	SourceMCP   Source = "mcp"
)

// Entry 是应用组装阶段注册进 Catalog 的内部记录。
// Tool 保留可执行实例；Source、Server 和 Enabled 是 AgentHub 自己的
// 管理信息，不复制 Tool.Info 中由工具提供的名称、描述或参数 Schema。
type Entry struct {
	Tool    tool.BaseTool
	Source  Source
	Server  string
	Enabled bool
}

// Item 是给 CLI 等诊断界面使用的只读展示模型。
// 它把 Tool.Info 返回的元数据与 Entry 中的管理信息合并，但不暴露
// 可执行 Tool 实例，避免展示层越过 Catalog 直接执行工具。
type Item struct {
	Name        string
	Source      Source
	Server      string
	Enabled     bool
	Description string
	Parameters  string
}

// Catalog 统一保存本地工具和 MCP 工具的注册条目。
// 它只负责校验、分类、启用筛选和元数据查询；真正的 ToolCall 选择与
// 执行仍由模型和 Eino ToolsNode 负责。
type Catalog struct {
	entries []Entry
}

// New 校验并复制注册条目，使 Catalog 不受调用方之后修改原切片的影响。
// 此处只校验 AgentHub 管理字段；名称、描述和参数 Schema 仍由 List
// 通过真实 Tool.Info 获取，避免维护第二份可能过期的工具元数据。
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

// EnabledTools 返回交给 Eino ReAct Agent / ToolsNode 的可执行工具集合。
// 禁用条目不会进入模型工具 Schema，也不能被 ToolsNode 路由；但它们仍
// 保留在 Catalog 中，供 List 和 /tools 解释当前配置状态。
func (c *Catalog) EnabledTools() []tool.BaseTool {
	enabled := make([]tool.BaseTool, 0, len(c.entries))

	for _, entry := range c.entries {
		if entry.Enabled {
			enabled = append(enabled, entry.Tool)
		}
	}

	return enabled
}

// List 返回全部注册条目的诊断信息，包括禁用工具。
// 工具名称、描述和参数 Schema 每次都从 Tool.Info 读取，Catalog 只补充
// 来源、所属 Server 和启用状态；读取或转换失败会终止本次列表构建。
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

		// ParamsOneOf 是 Eino 的参数定义；/tools 需要稳定的文本输出，
		// 因此先转为 JSON Schema，再编码成 JSON 字符串。无参数时用 {}。
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
			Server:      entry.Server,
			Enabled:     entry.Enabled,
			Description: info.Desc,
			Parameters:  parameters,
		})
	}

	return items, nil
}
