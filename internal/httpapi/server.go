package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
)

const address = "127.0.0.1:8080"

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Answer string `json:"answer"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// generator 是 HTTP Handler 所需的最小 Agent 能力。*react.Agent 会自然满足
// 该接口；测试可使用本地假实现，无需访问真实模型或 MCP Server。
type generator interface {
	Generate(
		ctx context.Context,
		input []*schema.Message,
		opts ...agent.AgentOption,
	) (*schema.Message, error)
}

func NewHandler(
	reactAgent generator,
	systemPrompt string,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /chat", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		var input chatRequest

		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			writeJSON(
				writer,
				http.StatusBadRequest,
				errorResponse{Error: "invalid JSON request"},
			)
			return
		}

		input.Message = strings.TrimSpace(input.Message)
		if input.Message == "" {
			writeJSON(
				writer,
				http.StatusBadRequest,
				errorResponse{Error: "message is required"},
			)
			return
		}

		message, err := reactAgent.Generate(
			request.Context(),
			[]*schema.Message{
				schema.SystemMessage(systemPrompt),
				schema.UserMessage(input.Message),
			},
		)
		if err != nil {
			log.Printf("execute HTTP chat: %v", err)

			writeJSON(
				writer,
				http.StatusBadGateway,
				errorResponse{Error: "agent execution failed"},
			)
			return
		}
		if message == nil {
			log.Printf("execute HTTP chat: agent returned nil message")

			writeJSON(
				writer,
				http.StatusBadGateway,
				errorResponse{Error: "agent execution failed"},
			)
			return
		}

		writeJSON(
			writer,
			http.StatusOK,
			chatResponse{Answer: message.Content},
		)
	})

	return mux
}

func writeJSON(
	writer http.ResponseWriter,
	status int,
	value any,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	writer.WriteHeader(status)

	if err := json.NewEncoder(writer).Encode(value); err != nil {
		log.Printf("encode HTTP response: %v", err)
	}
}

func Run(
	reactAgent generator,
	systemPrompt string,
) error {
	log.Printf(
		"HTTP server listening on http://%s",
		address,
	)

	return http.ListenAndServe(
		address,
		NewHandler(reactAgent, systemPrompt),
	)
}
