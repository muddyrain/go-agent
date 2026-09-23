package httpapi

import (
	"context"
	"log"

	"agenthub/internal/techproposal"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

// techProposalRequest 是 POST /tech-proposal 的请求体。
type techProposalRequest struct {
	Task string `json:"task"`
}

// techProposalResponse 是成功响应。
type techProposalResponse struct {
	Result string `json:"result"`
}

// registerTechProposalRoutes 注册技术方案生成相关路由。
func registerTechProposalRoutes(
	h *server.Hertz,
	workflow compose.Runnable[techproposal.ProposalRequest, techproposal.ProposalResult],
) {
	h.POST("/tech-proposal", func(ctx context.Context, c *app.RequestContext) {
		techProposalHandler(ctx, c, workflow)
	})
}

func techProposalHandler(
	ctx context.Context,
	c *app.RequestContext,
	workflow compose.Runnable[techproposal.ProposalRequest, techproposal.ProposalResult],
) {
	var req techProposalRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, errorResponse{Error: "invalid request body"})
		return
	}
	if req.Task == "" {
		c.JSON(400, errorResponse{Error: "task is required"})
		return
	}

	result, err := workflow.Invoke(ctx, techproposal.ProposalRequest{Task: req.Task})
	if err != nil {
		log.Printf("tech proposal workflow failed: %v", err)
		c.JSON(502, errorResponse{Error: "workflow execution failed"})
		return
	}

	c.JSON(200, techProposalResponse{Result: result.Content})
}
