package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

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
	DocumentID string `json:"document_id"`
	Chunks     int    `json:"chunks"`
	Message    string `json:"message"`
}

const (
	documentMaxChunkChars = 500
	documentOverlapChars  = 80
)

// generateDocumentID 为一次导入的原始文档生成逻辑 ID。
// 同一篇文档拆出的所有 Chunk 共享这个 ID。
func generateDocumentID() (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate document ID: %w", err)
	}

	return hex.EncodeToString(randomBytes), nil
}

// registerDocumentRoutes 注册文档相关的 HTTP 路由。
// 现在只有一个 POST /documents 接口，用来导入文档到知识库。
func registerDocumentRoutes(h *server.Hertz, store *knowledge.DocumentStore) {
	chunker, err := knowledge.NewChunker(
		documentMaxChunkChars,
		documentOverlapChars,
	)
	if err != nil {
		panic(fmt.Sprintf("create document chunker: %v", err))
	}

	h.POST("/documents", func(ctx context.Context, c *app.RequestContext) {

		var req documentRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: "invalid JSON request",
			})
			return
		}

		if strings.TrimSpace(req.Content) == "" {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: "content is required",
			})
			return
		}

		chunks := chunker.Chunk(req.Content)

		documentID, err := generateDocumentID()
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{
				Error: "failed to generate document ID",
			})
			return
		}

		docs := make([]knowledge.Document, 0, len(chunks))

		for index, chunk := range chunks {
			chunkMetadata := make(
				map[string]any,
				len(req.Metadata)+2,
			)

			for key, value := range req.Metadata {
				chunkMetadata[key] = value
			}

			chunkMetadata["document_id"] = documentID
			chunkMetadata["chunk_index"] = index

			docs = append(docs, knowledge.Document{
				Content:  chunk,
				Metadata: chunkMetadata,
			})
		}

		if err := store.AddDocuments(ctx, docs); err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse{
				Error: "failed to add document: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, documentResponse{
			DocumentID: documentID,
			Chunks:     len(chunks),
			Message:    "document added successfully",
		})
	})
}
