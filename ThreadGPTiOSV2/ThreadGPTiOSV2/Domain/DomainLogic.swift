import Foundation

enum ThreadGPTConstants {
    static let messagePageSize = 10
    static let sessionPageSize = 20
    static let initialChatConfirmation = "Context set! Your assistant has been configured with this as its instructions. Send your next message to start chatting."
    static let defaultServerURL = "http://localhost:8000"
}

func createOptimisticUserMessage(_ content: String, sessionID: String? = nil) -> ChatMessage {
    ChatMessage(
        id: UUID().uuidString,
        sessionID: sessionID ?? "",
        role: .user,
        content: content,
        replyCount: nil,
        createdAt: ISO8601DateFormatter().string(from: Date())
    )
}

func getSessionLabel(_ session: ChatSession) -> String {
    if let name = session.name, !name.isEmpty, name != "New conversation" {
        return name
    }
    if let prompt = session.systemPrompt, !prompt.isEmpty {
        return prompt
    }
    return "New conversation"
}

func isFirstMessageSession(_ session: ChatSession?) -> Bool {
    session?.systemPrompt?.isEmpty != false
}

func chatInputPlaceholder(isFirstMessage: Bool) -> String {
    isFirstMessage ? "Set the context for your conversation..." : "Send a message..."
}

func renderConversationHistory(session: ChatSession?, history: ConversationHistoryPage) -> [ChatMessage] {
    let sessionID = session?.sessionID ?? ""
    var messages: [ChatMessage] = []

    if let systemPrompt = session?.systemPrompt, !systemPrompt.isEmpty {
        messages.append(ChatMessage(
            id: "system-prompt:\(sessionID)",
            sessionID: sessionID,
            role: .user,
            content: systemPrompt,
            replyCount: nil,
            createdAt: session?.createdAt ?? ""
        ))
        messages.append(ChatMessage(
            id: "system-prompt-confirmation:\(sessionID)",
            sessionID: sessionID,
            role: .assistant,
            content: ThreadGPTConstants.initialChatConfirmation,
            replyCount: nil,
            createdAt: session?.createdAt ?? ""
        ))
    }

    messages.append(contentsOf: renderConversationPreviews(history: history, sessionID: sessionID))
    return messages
}

func renderConversationPreviews(history: ConversationHistoryPage, sessionID: String) -> [ChatMessage] {
    history.conversations.flatMap { conversation in
        [
            ChatMessage(
                id: "\(conversation.conversationID):user",
                sessionID: sessionID.isEmpty ? conversation.sessionID : sessionID,
                role: .user,
                content: conversation.userMessage,
                replyCount: nil,
                createdAt: conversation.createdAt
            ),
            ChatMessage(
                id: conversation.conversationID,
                sessionID: sessionID.isEmpty ? conversation.sessionID : sessionID,
                role: .assistant,
                content: conversation.assistantMessage,
                replyCount: conversation.replyCount,
                createdAt: conversation.createdAt
            )
        ]
    }
}

