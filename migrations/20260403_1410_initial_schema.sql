-- ThreadGPT Supabase Schema
-- Run this in Supabase SQL Editor

CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  api_key_hash TEXT NOT NULL,
  assistant_id TEXT,
  system_prompt TEXT,
  name TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX sessions_api_key_hash_idx ON sessions (api_key_hash);

CREATE TABLE session_conversations (
  conversation_id TEXT PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ DEFAULT now()
);

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE session_conversations ENABLE ROW LEVEL SECURITY;
