package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"agenthub/internal/toolcatalog"

	"github.com/cloudwego/eino/schema"
)

type catalogOutputTestTool struct{}

func (*catalogOutputTestTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "calculator__add_numbers",
		Desc: "计算两个普通数字的和",
	}, nil
}

func TestPrintToolCatalogShowsSourceServerAndStatus(t *testing.T) {
	catalog, err := toolcatalog.New(toolcatalog.Entry{
		Tool:    &catalogOutputTestTool{},
		Source:  toolcatalog.SourceMCP,
		Server:  "calculator",
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("toolcatalog.New() error = %v", err)
	}

	var output bytes.Buffer
	if err := printToolCatalog(t.Context(), &output, catalog); err != nil {
		t.Fatalf("printToolCatalog() error = %v", err)
	}

	for _, want := range []string{
		"tools: 1",
		"name: calculator__add_numbers",
		"source: mcp",
		"server: calculator",
		"status: disabled",
		"description: 计算两个普通数字的和",
		"parameters: {}",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("printToolCatalog() output = %q, want %q", output.String(), want)
		}
	}
}
