-- documents 表：存储文档内容和向量，用于 RAG 检索
CREATE TABLE documents (
    id BIGSERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    embedding vector(1024) NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- HNSW 索引：加速相似度搜索，用余弦距离
CREATE INDEX documents_embedding_idx ON documents USING hnsw (embedding vector_cosine_ops);
