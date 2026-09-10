package toolcatalog

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type testTool struct {
	info *schema.ToolInfo
}

func (t *testTool) Info(context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func TestCatalogListsAllEntriesAndFiltersEnabledTools(t *testing.T) {
	localTool := &testTool{
		info: &schema.ToolInfo{
			Name: "read_project_file",
			Desc: "读取项目文件",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"path": {
					Type:     schema.String,
					Desc:     "项目内相对路径",
					Required: true,
				},
			}),
		},
	}
	disabledMCPTool := &testTool{
		info: &schema.ToolInfo{
			Name: "remote_search",
			Desc: "远程搜索",
		},
	}

	catalog, err := New(
		Entry{
			Tool:    localTool,
			Source:  SourceLocal,
			Enabled: true,
		},
		Entry{
			Tool:    disabledMCPTool,
			Source:  SourceMCP,
			Server:  "search",
			Enabled: false,
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	enabled := catalog.EnabledTools()
	if len(enabled) != 1 {
		t.Fatalf("EnabledTools() length = %d, want 1", len(enabled))
	}
	if enabled[0] != tool.BaseTool(localTool) {
		t.Fatal("EnabledTools() did not return the enabled local tool")
	}

	items, err := catalog.List(t.Context())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() length = %d, want 2", len(items))
	}
	if items[0].Name != "read_project_file" ||
		items[0].Source != SourceLocal ||
		!items[0].Enabled {
		t.Fatalf("List() first item = %+v, want enabled local tool", items[0])
	}
	if items[0].Parameters == "{}" {
		t.Fatalf("List() first item parameters = %q, want path schema", items[0].Parameters)
	}
	if items[1].Name != "remote_search" ||
		items[1].Source != SourceMCP ||
		items[1].Server != "search" ||
		items[1].Enabled {
		t.Fatalf("List() second item = %+v, want disabled MCP tool", items[1])
	}
}
