package knowledge

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// getTestDB 连接测试数据库，不可用时跳过测试。
func getTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db, err := pgxpool.New(context.Background(), "postgres:///agenthub_test?sslmode=disable")
	if err != nil {
		t.Skipf("connect to test database: %v", err)
	}
	if err := db.Ping(context.Background()); err != nil {
		t.Skipf("ping test database: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(context.Background(), "TRUNCATE TABLE documents RESTART IDENTITY CASCADE")
		db.Close()
	})
	return db
}

// mockEmbedder 是测试用的嵌入模型，返回固定的向量。
type mockEmbedder struct {
	vectors map[string][]float32
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		if vec, ok := m.vectors[text]; ok {
			// 把短向量补到 1024 维，满足 vector(1024) 的要求
			result[i] = padTo1024(vec)
		} else {
			result[i] = padTo1024([]float32{0.1, 0.2, 0.3})
		}
	}
	return result, nil
}

// padTo1024 把短向量补零到 1024 维。
// 测试只需要前 3 维有区分度，后面补零不影响相似度计算。
func padTo1024(vec []float32) []float32 {
	padded := make([]float32, 1024)
	copy(padded, vec)
	return padded
}

// EmbedderFunc 让测试可以用函数构造特定的嵌入结果。
type EmbedderFunc func(ctx context.Context, texts []string) ([][]float32, error)

func (f EmbedderFunc) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return f(ctx, texts)
}

func TestDocumentStore_AddDocumentsRollsBackWholeBatch(t *testing.T) {
	db := getTestDB(t)
	ctx := context.Background()

	store := NewDocumentStore(db, EmbedderFunc(func(_ context.Context, _ []string) ([][]float32, error) {
		return [][]float32{
			padTo1024([]float32{0.1, 0.2, 0.3}), // 第一条维度合法，INSERT 能成功
			{0.1, 0.2},                          // 第二条不是 1024 维，INSERT 会失败
		}, nil
	}))

	err := store.AddDocuments(ctx, []Document{
		{Content: "第一个片段"},
		{Content: "第二个片段"},
	})
	if err == nil {
		t.Fatal("AddDocuments() error = nil, want second INSERT to fail")
	}

	var count int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM documents").Scan(&count); err != nil {
		t.Fatalf("count documents after rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("documents after rollback = %d, want 0", count)
	}
}

func TestDocumentStore_AddAndSearch(t *testing.T) {
	db := getTestDB(t)

	// 用 mock 嵌入模型，避免测试时调用真实 API
	embedder := &mockEmbedder{
		vectors: map[string][]float32{
			"苹果手机":    {0.9, 0.1, 0.0},
			"iPhone":  {0.8, 0.15, 0.0},
			"香蕉":      {0.0, 0.9, 0.1},
			"苹果手机怎么样": {0.85, 0.12, 0.0},
		},
	}

	store := NewDocumentStore(db, embedder)
	ctx := context.Background()

	// 添加文档
	docs := []Document{
		{Content: "苹果手机", Metadata: map[string]any{"source": "test"}},
		{Content: "iPhone", Metadata: map[string]any{"source": "test"}},
		{Content: "香蕉", Metadata: map[string]any{"source": "test"}},
	}
	if err := store.AddDocuments(ctx, docs); err != nil {
		t.Fatalf("AddDocuments error: %v", err)
	}

	// 搜索：查询"苹果手机怎么样"，应该返回"苹果手机"和"iPhone"
	results, err := store.Search(ctx, "苹果手机怎么样", 3)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// 第一个结果应该是最相关的
	if results[0].Document.Content != "苹果手机" && results[0].Document.Content != "iPhone" {
		t.Errorf("first result should be 苹果手机 or iPhone, got %s", results[0].Document.Content)
	}

	// "香蕉"的分数应该最低
	if results[2].Document.Content != "香蕉" {
		t.Errorf("last result should be 香蕉, got %s", results[2].Document.Content)
	}

	// 分数应该在 0-1 之间
	for _, r := range results {
		if r.Score < 0 || r.Score > 1 {
			t.Errorf("score should be between 0 and 1, got %f", r.Score)
		}
	}
}
