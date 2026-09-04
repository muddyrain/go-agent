package main

import (
	"context"
	"encoding/json"
	"fmt"

	"agenthub/internal/tool"
)

type weatherArguments struct {
	City string `json:"city"`
}

func newWeatherTool() (*tool.Function, error) {
	return tool.NewFunction(
		tool.Definition{
			Name:        "get_weather",
			Description: "查询指定城市的演示天气",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"city": {
						"type": "string",
						"description": "要查询天气的城市"
					}
				},
				"required": ["city"],
				"additionalProperties": false
			}`),
		},
		func(
			ctx context.Context,
			arguments json.RawMessage,
		) (string, error) {
			if ctx.Err() != nil {
				return "", fmt.Errorf("context canceled: %w", ctx.Err())
			}
			var args weatherArguments
			err := json.Unmarshal(arguments, &args)

			if err != nil {
				return "", fmt.Errorf("decode weather arguments: %w", err)
			}
			return fmt.Sprintf("%s今天晴，25°C。", args.City), nil
		},
	)
}
