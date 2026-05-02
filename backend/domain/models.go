package domain

type Session struct {
	ID           string  `json:"id"`
	APIKeyHash   string  `json:"api_key_hash"`
	AssistantID  *string `json:"assistant_id"`
	SystemPrompt *string `json:"system_prompt"`
	Name         *string `json:"name"`
	CreatedAt    string  `json:"created_at"`
}

type Message struct {
	ID         string `json:"id"`
	SessionID  string `json:"session_id"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	ReplyCount int    `json:"reply_count"`
	CreatedAt  string `json:"created_at"`
}

type ConversationRef struct {
	ConversationID string `json:"conversation_id"`
	SessionID      string `json:"session_id"`
	CreatedAt      string `json:"created_at"`
}

type ConversationPreview struct {
	ConversationID   string `json:"conversation_id"`
	SessionID        string `json:"session_id"`
	UserMessage      string `json:"user_message"`
	AssistantMessage string `json:"assistant_message"`
	ReplyCount       int    `json:"reply_count"`
	CreatedAt        string `json:"created_at"`
}
