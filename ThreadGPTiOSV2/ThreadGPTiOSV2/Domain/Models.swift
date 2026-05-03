import Foundation
import SwiftUI

enum MessageRole: String, Codable, Equatable {
    case user
    case assistant
}

struct ChatMessage: Identifiable, Codable, Equatable {
    let id: String
    let sessionID: String
    let role: MessageRole
    var content: String
    var replyCount: Int?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case sessionID = "session_id"
        case role
        case content
        case replyCount = "reply_count"
        case createdAt = "created_at"
    }

    var canStartThread: Bool {
        role == .assistant && !isSystemPromptConfirmation && !isStreaming
    }

    var isAssistant: Bool { role == .assistant }
    var isSystemPrompt: Bool { id.hasPrefix("system-prompt:") }
    var isSystemPromptConfirmation: Bool { id.hasPrefix("system-prompt-confirmation:") }
    var isStreaming: Bool { id == "__streaming__" }
}

struct ConversationPreview: Codable, Equatable {
    let conversationID: String
    let sessionID: String
    let userMessage: String
    let assistantMessage: String
    let replyCount: Int
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case conversationID = "conversation_id"
        case sessionID = "session_id"
        case userMessage = "user_message"
        case assistantMessage = "assistant_message"
        case replyCount = "reply_count"
        case createdAt = "created_at"
    }
}

struct ChatSession: Identifiable, Codable, Equatable {
    let sessionID: String?
    let assistantID: String?
    var systemPrompt: String?
    var name: String?
    let isNew: Bool?
    let createdAt: String?

    var id: String { sessionID ?? UUID().uuidString }

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case assistantID = "assistant_id"
        case systemPrompt = "system_prompt"
        case name
        case isNew = "is_new"
        case createdAt = "created_at"
    }
}

struct ConversationHistoryPage: Codable, Equatable {
    let conversations: [ConversationPreview]
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case conversations
        case hasMore = "has_more"
    }
}

struct ThreadHistoryPage: Codable, Equatable {
    let messages: [ChatMessage]
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case messages
        case hasMore = "has_more"
    }
}

struct SessionListPage: Codable, Equatable {
    let sessions: [ChatSession]
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case sessions
        case hasMore = "has_more"
    }
}

struct AuthInfo: Codable, Equatable {
    let expiresAt: String

    enum CodingKeys: String, CodingKey {
        case expiresAt = "expires_at"
    }
}

enum StreamEvent: Equatable {
    case sessionID(String)
    case chunk(String)
}

enum ThemePreference: String, CaseIterable, Identifiable {
    case system
    case light
    case dark

    var id: String { rawValue }

    var title: String {
        switch self {
        case .system: return "System"
        case .light: return "Light"
        case .dark: return "Dark"
        }
    }

    var preferredColorScheme: ColorScheme? {
        switch self {
        case .system: return nil
        case .light: return .light
        case .dark: return .dark
        }
    }
}

