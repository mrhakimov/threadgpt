export interface Message {
  id: string
  session_id: string
  role: "user" | "assistant"
  content: string
  openai_thread_id?: string
  parent_message_id?: string
  created_at: string
}

export interface Session {
  session_id?: string
  assistant_id?: string
  system_prompt?: string
  name?: string
  is_new?: boolean
  created_at?: string
}
