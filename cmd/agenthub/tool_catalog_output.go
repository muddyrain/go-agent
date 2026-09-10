package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"agenthub/internal/toolcatalog"
)

func printToolCatalog(
	ctx context.Context,
	writer io.Writer,
	catalog *toolcatalog.Catalog,
) error {
	items, err := catalog.List(ctx)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}

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
