package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agenthub/internal/knowledge"
)

// documentTestEmbedder 返回固定的 1024 维向量，避免测试调用真实嵌入 API。
type documentTestEmbedder struct{}

func (documentTestEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = make([]float32, 1024)
		embeddings[i][0] = 1
	}
	return embeddings, nil
}

func TestDocumentHandlerRejectsBlankContent(t *testing.T) {
	db := getTestDB(t)
	store := knowledge.NewDocumentStore(db, documentTestEmbedder{})
	h := NewServer(
		fakeGenerator{},
		"system prompt",
		NewSessionManager(NewPostgresSessionRepository(db), 20, 30*time.Minute, 5*time.Minute),
		"127.0.0.1:8080",
		store,
	)

	resp := postJSONToEngine(t, h, "/documents", `{"content":"  \n\t "}`)
	if resp.Code != 400 {
		t.Fatalf("status = %d, want 400; body = %s", resp.Code, resp.Body.String())
	}
	if got := resp.Body.String(); got != `{"error":"content is required"}` {
		t.Fatalf("body = %q, want content-required error", got)
	}
}

func TestDocumentHandlerImportsOverlappingChunks(t *testing.T) {
	db := getTestDB(t)

	var tableName *string
	if err := db.QueryRow(context.Background(), `SELECT to_regclass('public.documents')::text`).Scan(&tableName); err != nil {
		t.Fatalf("check documents table: %v", err)
	}
	if tableName == nil {
		t.Skip("documents table is unavailable; run database migrations for agenthub_test")
	}

	if _, err := db.Exec(context.Background(), "TRUNCATE TABLE documents RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate documents: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(context.Background(), "TRUNCATE TABLE documents RESTART IDENTITY")
	})

	store := knowledge.NewDocumentStore(db, documentTestEmbedder{})
	h := NewServer(
		fakeGenerator{},
		"system prompt",
		NewSessionManager(NewPostgresSessionRepository(db), 20, 30*time.Minute, 5*time.Minute),
		"127.0.0.1:8080",
		store,
	)

	content := strings.Repeat("甲", 600)
	requestBody, err := json.Marshal(documentRequest{
		Content: content,
		Metadata: map[string]any{
			"source":      "handbook.md",
			"document_id": "user-supplied-value",
			"chunk_index": 999,
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	resp := postJSONToEngine(t, h, "/documents", string(requestBody))
	if resp.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", resp.Code, resp.Body.String())
	}

	var response documentResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.DocumentID) != 32 {
		t.Fatalf("document ID length = %d, want 32", len(response.DocumentID))
	}
	if response.Chunks != 2 {
		t.Fatalf("response chunks = %d, want 2", response.Chunks)
	}

	rows, err := db.Query(
		context.Background(),
		`SELECT content, metadata->>'document_id', (metadata->>'chunk_index')::int, metadata->>'source'
		 FROM documents
		 ORDER BY (metadata->>'chunk_index')::int`,
	)
	if err != nil {
		t.Fatalf("query imported chunks: %v", err)
	}
	defer rows.Close()

	type storedChunk struct {
		content    string
		documentID string
		index      int
		source     string
	}

	var got []storedChunk
	for rows.Next() {
		var chunk storedChunk
		if err := rows.Scan(&chunk.content, &chunk.documentID, &chunk.index, &chunk.source); err != nil {
			t.Fatalf("scan imported chunk: %v", err)
		}
		got = append(got, chunk)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate imported chunks: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("stored chunks = %d, want 2", len(got))
	}
	if got[0].documentID != response.DocumentID || got[1].documentID != response.DocumentID {
		t.Fatalf("stored document IDs = %q, %q; want %q", got[0].documentID, got[1].documentID, response.DocumentID)
	}
	if got[0].index != 0 || got[1].index != 1 {
		t.Fatalf("stored chunk indexes = %d, %d; want 0, 1", got[0].index, got[1].index)
	}
	if got[0].source != "handbook.md" || got[1].source != "handbook.md" {
		t.Fatalf("stored sources = %q, %q; want handbook.md", got[0].source, got[1].source)
	}
	if len([]rune(got[0].content)) != 500 || len([]rune(got[1].content)) != 180 {
		t.Fatalf(
			"stored chunk lengths = %d, %d; want 500, 180",
			len([]rune(got[0].content)),
			len([]rune(got[1].content)),
		)
	}
}
