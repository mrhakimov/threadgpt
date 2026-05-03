import Foundation

@MainActor
final class ChatViewModel: ObservableObject {
    static let systemPromptConfirmation =
        "Context set! Your assistant has been configured with this as its instructions. Send your next message to start chatting."

    @Published var messages: [ConversationPreview] = []
    @Published var streamingContent = ""
    @Published var isStreaming = false
    @Published var isSending = false
    @Published var isLoading = false
    @Published var error: String?
    @Published var session: Session?
    @Published var hasMore = false

    var onUnauthorized: (() -> Void)?

    private let chatRepo: ChatRepository
    private let sessionRepo: SessionRepository
    private var currentSessionId: String?
    private var activeLoadID = UUID()

    init(
        chatRepo: ChatRepository = AppContainer.shared.chatRepo,
        sessionRepo: SessionRepository = AppContainer.shared.sessionRepo
    ) {
        self.chatRepo = chatRepo
        self.sessionRepo = sessionRepo
    }

    var displayMessages: [Message] {
        var result: [Message] = []
        let sessionId = session?.sessionId ?? currentSessionId ?? ""

        if let systemPrompt = session?.systemPrompt, !systemPrompt.isEmpty {
            result.append(Message(
                id: "system-prompt:\(sessionId)",
                sessionId: sessionId,
                role: .user,
                content: systemPrompt,
                replyCount: 0,
                createdAt: session?.createdAt ?? ""
            ))
            result.append(Message(
                id: "system-prompt-confirmation:\(sessionId)",
                sessionId: sessionId,
                role: .assistant,
                content: Self.systemPromptConfirmation,
                replyCount: 0,
                createdAt: session?.createdAt ?? ""
            ))
        }

        for conv in messages {
            result.append(Message(
                id: "\(conv.conversationId):user",
                sessionId: conv.sessionId,
                role: .user,
                content: conv.userMessage,
                replyCount: 0,
                createdAt: conv.createdAt
            ))
            result.append(Message(
                id: conv.conversationId,
                sessionId: conv.sessionId,
                role: .assistant,
                content: conv.assistantMessage,
                replyCount: conv.replyCount,
                createdAt: conv.createdAt
            ))
        }
        return result
    }

    var conversationName: String? {
        session?.name ?? session?.systemPrompt
    }

    var isFirstMessage: Bool {
        session?.systemPrompt == nil && messages.isEmpty
    }

    // MARK: - Load

    func loadSession(_ sessionId: String?) async {
        let loadID = UUID()
        activeLoadID = loadID
        currentSessionId = sessionId
        error = nil
        session = nil
        messages = []
        hasMore = false

        guard let sessionId, !sessionId.isEmpty else {
            isLoading = false
            return
        }

        isLoading = true
        defer {
            if activeLoadID == loadID {
                isLoading = false
            }
        }

        do {
            async let sessionTask = sessionRepo.getSession(id: sessionId)
            async let historyTask = chatRepo.fetchHistory(sessionId: sessionId, limit: 20, offset: 0)
            let (s, h) = try await (sessionTask, historyTask)
            guard activeLoadID == loadID else { return }
            session = s
            messages = h.conversations
            hasMore = h.hasMore
        } catch let appError as AppError where appError.isUnauthorized {
            guard activeLoadID == loadID else { return }
            onUnauthorized?()
        } catch {
            guard activeLoadID == loadID else { return }
            self.error = error.localizedDescription
        }
    }

    func loadMore() async {
        guard hasMore, !isLoading else { return }
        isLoading = true
        do {
            let page = try await chatRepo.fetchHistory(
                sessionId: currentSessionId, limit: 20, offset: messages.count
            )
            let older = page.conversations
            messages.insert(contentsOf: older, at: 0)
            hasMore = page.hasMore
        } catch let appError as AppError where appError.isUnauthorized {
            onUnauthorized?()
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    // MARK: - Send

    func sendMessage(_ content: String) async -> String? {
        guard !content.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return nil }

        let trimmed = content.trimmingCharacters(in: .whitespacesAndNewlines)
        error = nil
        isSending = true
        isStreaming = true
        streamingContent = ""

        // If first message (system prompt), no optimistic message
        let isSettingPrompt = isFirstMessage

        do {
            let resolvedId = try await chatRepo.sendMessage(
                userMessage: trimmed,
                sessionId: currentSessionId,
                forceNew: currentSessionId == nil
            ) { [weak self] chunk in
                Task { @MainActor in
                    self?.streamingContent += chunk
                }
            }

            isStreaming = false
            streamingContent = ""
            isSending = false

            // Resolve session
            let sid = resolvedId ?? currentSessionId
            if let sid {
                currentSessionId = sid
                // Refresh data
                let page = try await chatRepo.fetchHistory(sessionId: sid, limit: 20, offset: 0)
                messages = page.conversations
                hasMore = page.hasMore

                if isSettingPrompt || session == nil {
                    session = try? await sessionRepo.getSession(id: sid)
                }
            }

            return sid
        } catch let appError as AppError where appError.isUnauthorized {
            onUnauthorized?()
            isSending = false
            isStreaming = false
            return nil
        } catch {
            self.error = error.localizedDescription
            isSending = false
            isStreaming = false
            streamingContent = ""
            return nil
        }
    }

    func incrementReplyCount(for conversationId: String) {
        if let idx = messages.firstIndex(where: { $0.conversationId == conversationId }) {
            messages[idx].replyCount += 1
        }
    }

    func updateSystemPrompt(_ prompt: String) async -> Bool {
        let trimmed = prompt.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let sid = currentSessionId, !trimmed.isEmpty else { return false }
        error = nil

        do {
            try await sessionRepo.updateSession(id: sid, name: nil, systemPrompt: trimmed)
            session?.systemPrompt = trimmed
            return true
        } catch let appError as AppError where appError.isUnauthorized {
            onUnauthorized?()
            return false
        } catch {
            self.error = error.localizedDescription
            return false
        }
    }
}
