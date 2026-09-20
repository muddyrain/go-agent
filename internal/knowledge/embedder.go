// Package knowledge 提供知识库与 RAG 检索能力。
package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Embedder 把文本转成向量的接口。
// 抽象成接口方便替换实现（OpenAI、本地模型、Mock），也方便测试。
type Embedder interface {
	// Embed 把一批文本转成向量，返回和 texts 等长的向量切片。
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// OpenAIEmbedder 用 OpenAI 兼容的嵌入 API 实现 Embedder。
type OpenAIEmbedder struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewOpenAIEmbedder 创建 OpenAI 嵌入模型实例。
func NewOpenAIEmbedder(apiKey, baseURL, model string) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

// embeddingRequest 嵌入 API 的请求体。
type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embeddingResponse 嵌入 API 的响应体。
type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// Embed 调用 OpenAI 嵌入 API，把文本转成向量。
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// 1. 构造请求体
	reqBody := embeddingRequest{
		Model: e.model,
		Input: texts,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	// 2. 发 HTTP 请求到 /v1/embeddings
	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding API: %w", err)
	}
	defer resp.Body.Close()

	// 3. 读响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// 4. 解析 JSON
	var result embeddingResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal embedding response: %w", err)
	}

	// 5. 按 index 排序，保证返回顺序和输入一致
	embeddings := make([][]float32, len(texts))
	for _, item := range result.Data {
		if item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	return embeddings, nil
}
