-- 000001_create_sessions.down.sql
-- 回滚：删除 messages 和 sessions 表（注意顺序：先删子表，再删父表）

DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS sessions;
