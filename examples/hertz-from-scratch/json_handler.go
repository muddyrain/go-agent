package main

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

type greetRequest struct {
	Name string `json:"name"`
}

type greetResponse struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

func greetHandler(_ context.Context, c *app.RequestContext) {
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

	// 当前非流式模式下正文已经接收；这是业务大小检查，不是读取过程限内存。
	if len(body) > 4*1024 {
		c.String(http.StatusRequestEntityTooLarge, "request body too large\n")
		return
	}
	var req greetRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.String(http.StatusBadRequest, "invalid JSON request\n")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.String(http.StatusBadRequest, "name is required\n")
		return
	}
	response := greetResponse{
		Message: "Hello, " + name,
		Name:    name,
	}

	c.JSON(http.StatusOK, response)
}
