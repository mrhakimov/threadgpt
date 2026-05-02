export interface Message {
  id: string
  session_id: string
  role: "user" | "assistant"
  content: string
  reply_count?: number
  created_at: string
}

export interface ConversationPreview {
  conversation_id: string
  session_id: string
  user_message: string
  assistant_message: string
  reply_count: number
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

export interface ConversationHistoryPage {
  conversations: ConversationPreview[]
  has_more: boolean
}

export interface HistoryPage {
  messages: Message[]
  has_more: boolean
}
