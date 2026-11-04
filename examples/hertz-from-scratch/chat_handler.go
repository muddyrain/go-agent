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
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

type chatResponse struct {
	Answer    string `json:"answer"`
	SessionID string `json:"session_id,omitempty"`
	Turn      int    `json:"turn,omitempty"`
}

func newChatHandler(sessions *sessionStore) app.HandlerFunc {
	// 返回的Handler持有同一个store；每次请求只新建局部请求与响应数据。
	return func(_ context.Context, c *app.RequestContext) {
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
		sessionID := strings.TrimSpace(req.SessionID)

		answer, err := replyToMessage(message)
		if err != nil {
			// 当前处理函数只返回可公开的指令输入错误，不用于映射未来内部故障。
			c.String(http.StatusBadRequest, "%s\n", err.Error())
			return
		}
		response := chatResponse{
			Answer: answer,
		}
		if sessionID != "" {
			// 只在处理成功后计数；计数不代表客户端一定收到响应。
			response.SessionID = sessionID
			response.Turn = sessions.nextTurn(sessionID)
		}

		c.JSON(http.StatusOK, response)
	}
}
