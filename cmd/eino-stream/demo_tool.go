package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

type weatherArguments struct {
	City string `json:"city" jsonschema:"required,description=要查询天气的城市"`
}

func newWeatherTool() (tool.InvokableTool, error) {
	return toolutils.InferTool(
		"get_weather",
		"查询指定城市的演示天气",
		func(
			ctx context.Context,
			input weatherArguments,
		) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", fmt.Errorf("context canceled: %w", err)
			}

			return fmt.Sprintf(
				"%s今天晴，25°C。",
				input.City,
			), nil
		},
	)
}
