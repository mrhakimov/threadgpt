-- Roll back session conversation listing index

DROP INDEX IF EXISTS session_conversations_session_id_created_at_idx;
