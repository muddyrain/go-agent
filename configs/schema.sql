-- sessions：一个会话一行
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    last_access TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- messages：一条消息一行，属于某个 session
CREATE TABLE IF NOT EXISTS messages (
    id           BIGSERIAL PRIMARY KEY,
    session_id   TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role         TEXT NOT NULL,
    content      TEXT NOT NULL DEFAULT '',
    message_json JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 按 session_id + created_at 查询时加速
CREATE INDEX IF NOT EXISTS idx_messages_session_created
    ON messages(session_id, created_at);
