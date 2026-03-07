-- ThreadGPT Supabase Schema
-- Run this in Supabase SQL Editor

-- Identifies a "conversation session" by hashed API key
-- Multiple sessions per api_key_hash are allowed (one per conversation)
CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  api_key_hash TEXT NOT NULL,              -- SHA-256 of the user's OpenAI API key
  assistant_id TEXT,                        -- OpenAI Assistant ID (set after first message)
  system_prompt TEXT,                       -- The first message text
  name TEXT,                                -- User-facing conversation name
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX sessions_api_key_hash_idx ON sessions (api_key_hash);

-- Main thread messages (display only — each has its own OpenAI thread)
CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID REFERENCES sessions(id),
  role TEXT NOT NULL,                   -- 'user' | 'assistant'
  content TEXT NOT NULL,
  openai_thread_id TEXT,               -- The OpenAI Thread used for this exchange
  parent_message_id UUID REFERENCES messages(id),  -- set for sub-thread messages
  created_at TIMESTAMPTZ DEFAULT now()
);

-- Disable RLS for simplicity (add policies if needed)
ALTER TABLE sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE messages DISABLE ROW LEVEL SECURITY;

-- Migration: if upgrading from the old schema with UNIQUE constraint on api_key_hash:
-- ALTER TABLE sessions DROP CONSTRAINT sessions_api_key_hash_key;
-- ALTER TABLE sessions ADD COLUMN IF NOT EXISTS name TEXT;
