package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"agenthub/internal/toolcatalog"
)

// printToolCatalog 是 /tools 的 CLI 展示层：它只把 Catalog.List 返回的
// Item 格式化到 writer，不筛选启用状态、不读取底层 Tool，也不执行 ToolCall。
func printToolCatalog(
	ctx context.Context,
	writer io.Writer,
	catalog *toolcatalog.Catalog,
) error {
	items, err := catalog.List(ctx)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}

	// 先在内存中构造完整输出，再一次写入 writer，避免元数据转换中途
	// 失败时向终端留下半份工具目录。
	var output strings.Builder

	fmt.Fprintf(&output, "tools: %d\n", len(items))

	for _, item := range items {
		status := "disabled"
		if item.Enabled {
			status = "enabled"
		}

		fmt.Fprintf(
			&output,
			"- name: %s\n"+
				"  source: %s\n"+
				"  status: %s\n",
			item.Name,
			item.Source,
			status,
		)

		// Server 只对 MCP 工具有意义；本地工具保持空值，因此不打印
		// 空字段，避免把“无所属 Server”误解成配置缺失。
		if item.Server != "" {
			fmt.Fprintf(
				&output,
				"  server: %s\n",
				item.Server,
			)
		}

		fmt.Fprintf(
			&output,
			"  description: %s\n"+
				"  parameters: %s\n",
			item.Description,
			item.Parameters,
		)
	}

	if _, err := io.WriteString(writer, output.String()); err != nil {
		return fmt.Errorf("write tool catalog: %w", err)
	}

	return nil
}
