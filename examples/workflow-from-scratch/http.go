package main

import (
	"context"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

type proposalHTTPRequest struct {
	Task string `json:"task"`
}

type proposalHTTPResponse struct {
	Result string `json:"result"`
}

func newProposalHandler(workflow compose.Runnable[proposalRequest, proposalResult]) app.HandlerFunc {
	// 闭包持有已构建的Runnable；每次请求独立解析req和执行任务。
	return func(ctx context.Context, c *app.RequestContext) {
		mediaType, _, err := mime.ParseMediaType(string(c.GetHeader("Content-Type")))
		if err != nil || mediaType != "application/json" {
			c.String(
				http.StatusUnsupportedMediaType,
				"Content-Type must be application/json\n",
			)
			return
		}
		body := c.Request.Body()
		// Hertz已接收正文；此处是业务大小检查，不是读取过程限内存。
		if len(body) > 4*1024 {
			c.String(
				http.StatusRequestEntityTooLarge,
				"request body too large\n",
			)
			return
		}
		var req proposalHTTPRequest
		if err := json.Unmarshal(body, &req); err != nil {
			c.String(http.StatusBadRequest, "invalid JSON request\n")
			return
		}
		task := strings.TrimSpace(req.Task)
		if task == "" {
			c.String(http.StatusBadRequest, "task is required\n")
			return
		}
		// 使用当前请求ctx，而非启动时Background；执行错误不终止整个服务。
		result, err := workflow.Invoke(ctx, proposalRequest{Task: task})
		if err != nil {
			log.Printf("proposal workflow failed: %v", err)
			c.String(
				http.StatusInternalServerError,
				"workflow execution failed\n",
			)
			return
		}

		c.JSON(
			http.StatusOK,
			proposalHTTPResponse{Result: result.Content},
		)
	}
}

func newProposalServer(workflow compose.Runnable[proposalRequest, proposalResult]) *server.Hertz {
	// 独立教学服务使用Default的内置recovery，不复制8091的自定义中间件。
	h := server.Default(
		server.WithHostPorts("127.0.0.1:8092"),
		server.WithHandleMethodNotAllowed(true),
		server.WithSenseClientDisconnection(true),
	)

	h.POST("/tech-proposal", newProposalHandler(workflow))
	return h
}
