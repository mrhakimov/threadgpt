import Foundation

struct ChatStateSnapshot {
    let messages: [ChatMessage]
    let hasMoreMessages: Bool
    let session: ChatSession?
    let loadedConversationCount: Int
}

struct ChatLoadResult {
    let snapshot: ChatStateSnapshot
    let resolvedSessionID: String?
}

struct SendChatTurnResult {
    let snapshot: ChatStateSnapshot
    let resolvedSessionID: String?
}

struct ChatPageLoadResult {
    let messages: [ChatMessage]
    let hasMore: Bool
    let loadedConversationCount: Int
}

final class ChatService {
    private let sessionRepository: SessionRepository
    private let chatRepository: ChatRepository

    init(sessionRepository: SessionRepository, chatRepository: ChatRepository) {
        self.sessionRepository = sessionRepository
        self.chatRepository = chatRepository
    }

    func loadChatSession(sessionID: String?) async throws -> ChatLoadResult {
        guard let sessionID else {
            return ChatLoadResult(
                snapshot: ChatStateSnapshot(messages: [], hasMoreMessages: false, session: nil, loadedConversationCount: 0),
                resolvedSessionID: nil
            )
        }

        async let history = sessionRepository.fetchHistory(
            sessionID: sessionID,
            limit: ThreadGPTConstants.messagePageSize,
            offset: 0
        )
        async let session = sessionRepository.fetchSession(sessionID: sessionID)

        let resolvedHistory = try await history
        let resolvedSession = try await session
        return ChatLoadResult(
            snapshot: ChatStateSnapshot(
                messages: renderConversationHistory(session: resolvedSession, history: resolvedHistory),
                hasMoreMessages: resolvedHistory.hasMore,
                session: ChatSession(
                    sessionID: sessionID,
                    assistantID: resolvedSession.assistantID,
                    systemPrompt: resolvedSession.systemPrompt,
                    name: resolvedSession.name,
                    isNew: false,
                    createdAt: resolvedSession.createdAt
                ),
                loadedConversationCount: resolvedHistory.conversations.count
            ),
            resolvedSessionID: nil
        )
    }

    func loadOlderMessages(session: ChatSession, loadedConversationCount: Int) async throws -> ChatPageLoadResult {
        let history = try await sessionRepository.fetchHistory(
            sessionID: session.sessionID,
            limit: ThreadGPTConstants.messagePageSize,
            offset: loadedConversationCount
        )
        return ChatPageLoadResult(
            messages: renderConversationPreviews(history: history, sessionID: session.sessionID ?? ""),
            hasMore: history.hasMore,
            loadedConversationCount: history.conversations.count
        )
    }

    func loadCompleteHistory(session: ChatSession) async throws -> ChatPageLoadResult {
        let history = try await sessionRepository.fetchHistory(sessionID: session.sessionID, limit: 10_000, offset: 0)
        return ChatPageLoadResult(
            messages: renderConversationHistory(session: session, history: history),
            hasMore: false,
            loadedConversationCount: history.conversations.count
        )
    }

    func sendChatTurn(
        content: String,
        requestedSessionID: String?,
        currentSession: ChatSession?,
        onChunk: @escaping @MainActor (String) -> Void
    ) async throws -> SendChatTurnResult {
        let forceNew = requestedSessionID == nil
        let activeSessionID = forceNew ? nil : (requestedSessionID ?? currentSession?.sessionID)
        var accumulated = ""
        var streamedSessionID: String?

        for try await event in chatRepository.sendChatMessage(
            userMessage: content,
            sessionID: activeSessionID,
            forceNew: forceNew
        ) {
            switch event {
            case .sessionID(let sessionID):
                streamedSessionID = sessionID
            case .chunk(let chunk):
                accumulated += chunk
                await onChunk(accumulated)
            }
        }

        let resolvedSessionID = streamedSessionID ?? activeSessionID
        let history = try await sessionRepository.fetchHistory(
            sessionID: resolvedSessionID,
            limit: ThreadGPTConstants.messagePageSize,
            offset: 0
        )

        let needsFreshSession =
            resolvedSessionID != nil &&
            (resolvedSessionID != (requestedSessionID ?? currentSession?.sessionID) || currentSession?.systemPrompt == nil)

        let fetchedSession: ChatSession?
        if needsFreshSession, let resolvedSessionID {
            fetchedSession = try await sessionRepository.fetchSession(sessionID: resolvedSessionID)
        } else {
            fetchedSession = currentSession
        }

        let normalizedSession = fetchedSession.map {
            ChatSession(
                sessionID: resolvedSessionID ?? $0.sessionID,
                assistantID: $0.assistantID,
                systemPrompt: $0.systemPrompt,
                name: $0.name,
                isNew: false,
                createdAt: $0.createdAt
            )
        }

        return SendChatTurnResult(
            snapshot: ChatStateSnapshot(
                messages: renderConversationHistory(session: normalizedSession, history: history),
                hasMoreMessages: history.hasMore,
                session: normalizedSession,
                loadedConversationCount: history.conversations.count
            ),
            resolvedSessionID: resolvedSessionID
        )
    }
}
