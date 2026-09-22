package knowledge

import (
	"context"
	"strings"
	"testing"
)

func TestSearchToolReturnsSourceMetadataToModel(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	embedder := &mockEmbedder{
		vectors: map[string][]float32{
			"年假有几天": {1, 0, 0},
			"年假规定":  {1, 0, 0},
			"补充说明":  {0.9, 0.1, 0},
		},
	}
	store := NewDocumentStore(db, embedder)

	if err := store.AddDocuments(ctx, []Document{
		{
			Content: "年假规定",
			Metadata: map[string]any{
				"source":      "employee-handbook.md",
				"document_id": "doc-123",
				"chunk_index": 2,
			},
		},
		{
			Content:  "补充说明",
			Metadata: map[string]any{},
		},
	}); err != nil {
		t.Fatalf("AddDocuments() error = %v", err)
	}

	searchTool, err := NewSearchTool(store)
	if err != nil {
		t.Fatalf("NewSearchTool() error = %v", err)
	}

	result, err := searchTool.InvokableRun(ctx, `{"query":"年假有几天"}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}

	for _, want := range []string{
		"检索到 2 条相关文档",
		"来源：employee-handbook.md",
		"文档 ID：doc-123",
		"分块序号：2",
		"内容：\n年假规定",
		"来源：来源未知",
		"内容：\n补充说明",
		"相似度：",
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("tool result does not contain %q; result = %q", want, result)
		}
	}
}

func TestMetadataIntSupportsJSONNumberAndRejectsInvalidValue(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
		want     int
		wantOK   bool
	}{
		{
			name:     "integer",
			metadata: map[string]any{"chunk_index": 2},
			want:     2,
			wantOK:   true,
		},
		{
			name:     "JSON number",
			metadata: map[string]any{"chunk_index": float64(3)},
			want:     3,
			wantOK:   true,
		},
		{
			name:     "wrong type",
			metadata: map[string]any{"chunk_index": "4"},
			wantOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := metadataInt(tt.metadata, "chunk_index")
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("metadataInt() = (%d, %t), want (%d, %t)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
