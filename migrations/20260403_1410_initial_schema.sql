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

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID REFERENCES sessions(id),
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  openai_thread_id TEXT,
  parent_message_id UUID REFERENCES messages(id),
  created_at TIMESTAMPTZ DEFAULT now()
);

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
