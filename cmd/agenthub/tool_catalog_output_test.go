package main

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
		Name: "read_project_file",
		Desc: "读取项目文件",
	}, nil
}

func TestPrintToolCatalogShowsSourceAndStatus(t *testing.T) {
	catalog, err := toolcatalog.New(toolcatalog.Entry{
		Tool:    &catalogOutputTestTool{},
		Source:  toolcatalog.SourceLocal,
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
		"name: read_project_file",
		"source: local",
		"status: disabled",
		"description: 读取项目文件",
		"parameters: {}",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("printToolCatalog() output = %q, want %q", output.String(), want)
		}
	}
}
