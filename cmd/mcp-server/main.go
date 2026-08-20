package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// 简单的 MCP Server，提供 echo 工具

func main() {

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg map[string]interface{}

		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		method, _ := msg["method"].(string)
		id := msg["id"]

		switch method {
		case "initialize":
			respond(id, map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]string{
					"name":    "echo-server",
					"version": "0.1",
				},
			})
		case "tools/list":
			respond(id, map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "echo",
						"description": "回显输入的文本",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"text": map[string]interface{}{
									"type":        "string",
									"description": "要回显的文本",
								},
							},
							"required": []string{"text"},
						},
					},
				},
			})

		case "tools/call":
			params, _ := msg["params"].(map[string]interface{})
			name, _ := params["name"].(string)
			args, _ := params["arguments"].(map[string]interface{})

			if name == "echo" {
				text, _ := args["text"].(string)
				respond(id, map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": fmt.Sprintf("Echo: %s", text)},
					},
				})
			}

		case "notifications/initialized":
			// 通知，不需要响应
		}
	}
}

func respond(id interface{}, result interface{}) {
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}
