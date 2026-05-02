-- Improve session conversation listing by session and creation time

CREATE INDEX session_conversations_session_id_created_at_idx
ON session_conversations (session_id, created_at);
