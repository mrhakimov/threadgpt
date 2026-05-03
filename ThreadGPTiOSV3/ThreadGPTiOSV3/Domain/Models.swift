import Foundation

struct Session: Codable, Identifiable, Equatable {
    let sessionId: String?
    let assistantId: String?
    var systemPrompt: String?
    var name: String?
    let isNew: Bool?
    let createdAt: String?

    var id: String { sessionId ?? UUID().uuidString }

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case assistantId = "assistant_id"
        case systemPrompt = "system_prompt"
        case name
        case isNew = "is_new"
        case createdAt = "created_at"
    }
}

struct Message: Identifiable, Equatable {
    let id: String
    let sessionId: String
    let role: MessageRole
    let content: String
    var replyCount: Int
    let createdAt: String

    var isSystemPrompt: Bool {
        id.hasPrefix("system-prompt:")
    }

    var isSystemPromptConfirmation: Bool {
        id.hasPrefix("system-prompt-confirmation:")
    }

    enum MessageRole: String, Codable {
        case user
        case assistant
    }
}

extension Message: Codable {
    enum CodingKeys: String, CodingKey {
        case id
        case sessionId = "session_id"
        case role
        case content
        case replyCount = "reply_count"
        case createdAt = "created_at"
    }
}

struct ConversationPreview: Codable, Identifiable, Equatable {
    let conversationId: String
    let sessionId: String
    let userMessage: String
    let assistantMessage: String
    var replyCount: Int
    let createdAt: String

    var id: String { conversationId }

    enum CodingKeys: String, CodingKey {
        case conversationId = "conversation_id"
        case sessionId = "session_id"
        case userMessage = "user_message"
        case assistantMessage = "assistant_message"
        case replyCount = "reply_count"
        case createdAt = "created_at"
    }
}

struct ConversationHistoryPage: Codable {
    let conversations: [ConversationPreview]
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case conversations
        case hasMore = "has_more"
    }
}

struct SessionsPage: Codable {
    let sessions: [Session]
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case sessions
        case hasMore = "has_more"
    }
}

struct ThreadPage: Codable {
    let messages: [Message]
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case messages
        case hasMore = "has_more"
    }
}

struct AuthInfo: Codable {
    let expiresAt: String

    enum CodingKeys: String, CodingKey {
        case expiresAt = "expires_at"
    }
}

struct AuthResponse: Codable {
    let ok: Bool
    let token: String
}
