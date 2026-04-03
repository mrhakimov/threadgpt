export interface Message {
  id: string
  session_id: string
  role: "user" | "assistant"
  content: string
  parent_message_id?: string
  reply_count?: number
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

export interface HistoryPage {
  messages: Message[]
  has_more: boolean
}
