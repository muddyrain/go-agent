package httpapi

import (
	"context"
	"net/http"

	"agenthub/internal/knowledge"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

// documentRequest 是导入文档 API 的请求体。
type documentRequest struct {
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// documentResponse 是导入文档 API 的响应体。
type documentResponse struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

// registerDocumentRoutes 注册文档相关的 HTTP 路由。
// 现在只有一个 POST /documents 接口，用来导入文档到知识库。

func registerDocumentRoutes(h *server.Hertz, store *knowledge.DocumentStore) {
	h.POST("/documents", func(ctx context.Context, c *app.RequestContext) {
		var req documentRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: "invalid JSON request",
			})
			return
		}
	})
}
