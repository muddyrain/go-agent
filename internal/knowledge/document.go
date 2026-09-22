package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Document 表示一个可检索的文档片段。
type Document struct {
	ID        int64
	Content   string
	Metadata  map[string]any
	CreatedAt time.Time
}

// SearchResult 表示检索结果，包含文档和相似度分数。
type SearchResult struct {
	Document Document
	Score    float64 // 相似度分数，0-1，越大越相似
}

// DocumentStore 提供文档的向量化存储和相似度检索。
type DocumentStore struct {
	db       *pgxpool.Pool
	embedder Embedder
}

// NewDocumentStore 创建文档存储实例。
func NewDocumentStore(db *pgxpool.Pool, embedder Embedder) *DocumentStore {
	return &DocumentStore{
		db:       db,
		embedder: embedder,
	}
}

// AddDocuments 把一批文档转成向量并存入数据库。
func (s *DocumentStore) AddDocuments(ctx context.Context, docs []Document) error {
	if len(docs) == 0 {
		return nil
	}

	// 1. 提取所有文档内容，批量调用嵌入模型
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	embeddings, err := s.embedder.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed documents: %w", err)
	}

	// 2. 开启事务，保证一批文档要么全部写入，要么全部回滚。
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin add documents transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Commit 成功后再次 Rollback 是无害的。

	// 3. 在同一个事务中插入全部文档。
	for i, doc := range docs {
		embeddingStr := vectorToString(embeddings[i])

		var metadataJSON []byte
		if doc.Metadata != nil {
			metadataJSON, err = json.Marshal(doc.Metadata)
			if err != nil {
				return fmt.Errorf("marshal metadata: %w", err)
			}
		}

		_, err = tx.Exec(
			ctx,
			`INSERT INTO documents (content, embedding, metadata) VALUES ($1, $2::vector, $3)`,
			doc.Content,
			embeddingStr,
			metadataJSON,
		)
		if err != nil {
			return fmt.Errorf("insert document %d: %w", i, err)
		}
	}

	// 4. 只有全部 INSERT 成功后才提交事务。
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit add documents transaction: %w", err)
	}

	return nil
}

// Search 用相似度搜索找到和 query 最相关的 topK 个文档。
func (s *DocumentStore) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	// 1. 把查询转成向量
	embeddings, err := s.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}

	queryVector := vectorToString(embeddings[0])

	// 2. 相似度搜索
	// <=> 是余弦距离操作符，值越小越相似
	// 1 - (embedding <=> $2::vector) 把距离转成相似度分数（0-1，越大越相似）
	rows, err := s.db.Query(ctx,
		`SELECT id, content, metadata, created_at, 1 - (embedding <=> $2::vector) AS score
		 FROM documents
		 ORDER BY embedding <=> $2::vector
		 LIMIT $1`,
		topK, queryVector,
	)
	if err != nil {
		return nil, fmt.Errorf("search documents: %w", err)
	}
	defer rows.Close()

	// 3. 遍历结果集
	var results []SearchResult
	for rows.Next() {
		var doc Document
		var score float64
		var metadataBytes []byte

		err := rows.Scan(&doc.ID, &doc.Content, &metadataBytes, &doc.CreatedAt, &score)
		if err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}

		if len(metadataBytes) > 0 {
			if err := json.Unmarshal(metadataBytes, &doc.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		results = append(results, SearchResult{
			Document: doc,
			Score:    score,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}

	return results, nil
}

// vectorToString 把 float32 向量转成 pgvector 需要的字符串格式 '[0.1, 0.2, ...]'。
// pgvector 的向量类型不能直接传 []float32，需要传字符串。
func vectorToString(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, "%f", v)
	}
	sb.WriteByte(']')
	return sb.String()
}
