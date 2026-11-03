package main

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Answer string `json:"answer"`
}

func chatHandler(_ context.Context, c *app.RequestContext) {
	mediaType, _, err := mime.ParseMediaType(
		string(c.GetHeader("Content-Type")),
	)
	if err != nil || mediaType != "application/json" {
		c.String(
			http.StatusUnsupportedMediaType,
			"Content-Type must be application/json\n",
		)
		return
	}
	body := c.Request.Body()
	// 当前正文已接收；这是业务大小检查，不是读取过程限内存。
	if len(body) > 4*1024 {
		c.String(
			http.StatusRequestEntityTooLarge,
			"request body too large\n",
		)
		return
	}
	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.String(http.StatusBadRequest, "invalid JSON request\n")
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		c.String(http.StatusBadRequest, "message is required\n")
		return
	}
	response := chatResponse{
		Answer: "收到：" + message,
	}

	c.JSON(http.StatusOK, response)
}
