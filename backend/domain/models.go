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
	ID              string  `json:"id"`
	SessionID       string  `json:"session_id"`
	Role            string  `json:"role"`
	Content         string  `json:"content"`
	OpenAIThreadID  *string `json:"openai_thread_id"`
	ParentMessageID *string `json:"parent_message_id"`
	ReplyCount      int     `json:"reply_count"`
	CreatedAt       string  `json:"created_at"`
}
