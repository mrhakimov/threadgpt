import Foundation

struct APIErrorEnvelope: Decodable {
    let error: APIErrorDescriptor
}

struct APIErrorDescriptor: Decodable {
    let code: String
    let message: String
    let status: Int?
}

struct AuthRequest: Encodable {
    let apiKey: String
}

struct AuthResponse: Decodable {
    let ok: Bool
    let token: String
}

struct AuthInfo: Decodable {
    let expiresAt: String
}

struct EmptyRequest: Encodable {}

struct SessionInfo: Codable, Hashable, Identifiable {
    let sessionId: String?
    let assistantId: String?
    var systemPrompt: String?
    var name: String?
    let isNew: Bool?
    let createdAt: String?

    var id: String {
        sessionId ?? "new-session"
    }

    var displayTitle: String {
        let trimmedName = (name ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        if !trimmedName.isEmpty, trimmedName != "New conversation" {
            return trimmedName
        }

        let prompt = (systemPrompt ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        if !prompt.isEmpty {
            return prompt
        }

        return "New conversation"
    }
}

struct SessionListResponse: Decodable {
    let sessions: [SessionInfo]
    let hasMore: Bool
}

struct CreateSessionRequest: Encodable {
    let name: String
}

struct UpdateSessionRequest: Encodable {
    let name: String?
    let systemPrompt: String?
}

struct ChatRequest: Encodable {
    let userMessage: String
    let sessionId: String
    let forceNew: Bool
}

struct ThreadRequest: Encodable {
    let conversationId: String
    let userMessage: String
}

struct ConversationPreview: Codable, Hashable, Identifiable {
    let conversationId: String
    let sessionId: String
    let userMessage: String
    let assistantMessage: String
    let replyCount: Int
    let createdAt: String

    var id: String {
        conversationId
    }
}

struct ConversationHistoryPage: Decodable {
    let conversations: [ConversationPreview]
    let hasMore: Bool
}

struct Message: Codable, Hashable, Identifiable {
    let id: String
    let sessionId: String
    let role: String
    var content: String
    var replyCount: Int?
    let createdAt: String

    var isAssistant: Bool {
        role == "assistant"
    }

    var isUser: Bool {
        role == "user"
    }

    static func optimisticUser(_ content: String, sessionId: String?) -> Message {
        Message(
            id: "optimistic-user-\(UUID().uuidString)",
            sessionId: sessionId ?? "",
            role: "user",
            content: content,
            replyCount: nil,
            createdAt: ISO8601DateFormatter().string(from: Date())
        )
    }

    static func systemPrompt(_ content: String, sessionId: String, createdAt: String?) -> Message {
        Message(
            id: "system-prompt:\(sessionId)",
            sessionId: sessionId,
            role: "user",
            content: content,
            replyCount: nil,
            createdAt: createdAt ?? ""
        )
    }

    static func initialConfirmation(sessionId: String, createdAt: String?) -> Message {
        Message(
            id: "system-prompt-confirmation:\(sessionId)",
            sessionId: sessionId,
            role: "assistant",
            content: AppConfig.initialChatConfirmation,
            replyCount: nil,
            createdAt: createdAt ?? ""
        )
    }
}

struct MessageHistoryPage: Decodable {
    let messages: [Message]
    let hasMore: Bool
}

struct StreamEvent: Decodable {
    let sessionId: String?
    let chunk: String?
}
