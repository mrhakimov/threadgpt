import Foundation

@MainActor
final class ChatViewModel: ObservableObject {
    @Published var sessions: [SessionInfo] = []
    @Published var selectedSessionId: String?
    @Published var session: SessionInfo?
    @Published var messages: [Message] = []
    @Published var streamingContent = ""
    @Published var errorMessage: String?
    @Published var isLoading = false
    @Published var isSending = false
    @Published var isLoadingSessions = false
    @Published var hasMoreMessages = false

    let appModel: AppViewModel

    private var loadedConversationCount = 0
    private var isDraftConversation = false
    private var didLoadInitialState = false

    init(appModel: AppViewModel) {
        self.appModel = appModel
    }

    var title: String {
        session?.displayTitle ?? "ThreadGPT"
    }

    var isFirstMessage: Bool {
        let prompt = session?.systemPrompt?.trimmingCharacters(in: .whitespacesAndNewlines)
        return prompt?.isEmpty != false
    }

    func loadInitialStateIfNeeded() async {
        guard !didLoadInitialState else {
            return
        }

        didLoadInitialState = true
        await loadInitialState()
    }

    func loadInitialState() async {
        isLoading = true
        errorMessage = nil

        do {
            let client = try appModel.makeClient()
            let currentSession = try await client.initSession()
            await refreshSessions()

            if currentSession.isNew == true || currentSession.sessionId == nil {
                selectedSessionId = nil
                session = currentSession
                messages = []
                hasMoreMessages = false
                loadedConversationCount = 0
                isDraftConversation = false
            } else {
                selectedSessionId = currentSession.sessionId
                let history = try await client.fetchHistory(sessionId: nil)
                session = currentSession
                messages = renderConversationHistory(session: currentSession, history: history)
                hasMoreMessages = history.hasMore
                loadedConversationCount = history.conversations.count
                isDraftConversation = false
            }
        } catch {
            await handle(error)
        }

        isLoading = false
    }

    func refreshSessions() async {
        isLoadingSessions = true

        do {
            let response = try await appModel.makeClient().fetchSessions()
            sessions = response.sessions
        } catch {
            errorMessage = appModel.errorMessage(from: error)
        }

        isLoadingSessions = false
    }

    func startNewConversation() {
        selectedSessionId = nil
        session = SessionInfo(
            sessionId: nil,
            assistantId: nil,
            systemPrompt: nil,
            name: nil,
            isNew: true,
            createdAt: nil
        )
        messages = []
        streamingContent = ""
        errorMessage = nil
        hasMoreMessages = false
        loadedConversationCount = 0
        isDraftConversation = true
    }

    func loadSession(_ sessionId: String) async {
        guard !isSending else {
            return
        }

        isLoading = true
        errorMessage = nil
        selectedSessionId = sessionId
        streamingContent = ""

        do {
            let client = try appModel.makeClient()
            async let history = client.fetchHistory(sessionId: sessionId)
            async let fetchedSession = client.fetchSession(sessionId: sessionId)

            let (historyPage, sessionInfo) = try await (history, fetchedSession)
            session = sessionInfo
            messages = renderConversationHistory(session: sessionInfo, history: historyPage)
            hasMoreMessages = historyPage.hasMore
            loadedConversationCount = historyPage.conversations.count
            isDraftConversation = false
        } catch {
            await handle(error)
        }

        isLoading = false
    }

    func sendMessage(_ content: String) async {
        let trimmed = content.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, !isSending else {
            return
        }

        isSending = true
        errorMessage = nil
        streamingContent = ""

        let forceNew = isDraftConversation
        let activeSessionId = forceNew ? nil : (selectedSessionId ?? session?.sessionId)
        let wasFirstMessage = isFirstMessage

        if !wasFirstMessage {
            messages.append(.optimisticUser(trimmed, sessionId: activeSessionId))
        }

        do {
            let client = try appModel.makeClient()
            var accumulated = ""

            let streamedSessionId = try await client.sendChatMessage(
                trimmed,
                sessionId: activeSessionId,
                forceNew: forceNew
            ) { chunk in
                accumulated += chunk
                self.streamingContent = accumulated
            }

            guard let resolvedSessionId = streamedSessionId ?? activeSessionId else {
                streamingContent = ""
                isSending = false
                return
            }

            let history = try await client.fetchHistory(sessionId: resolvedSessionId)
            let freshSession = try await client.fetchSession(sessionId: resolvedSessionId)

            selectedSessionId = resolvedSessionId
            session = freshSession
            messages = renderConversationHistory(session: freshSession, history: history)
            hasMoreMessages = history.hasMore
            loadedConversationCount = history.conversations.count
            streamingContent = ""
            isDraftConversation = false

            await refreshSessions()
        } catch {
            streamingContent = ""
            await handle(error)
        }

        isSending = false
    }

    func loadMoreMessages() async {
        guard hasMoreMessages, let sessionId = selectedSessionId ?? session?.sessionId else {
            return
        }

        do {
            let history = try await appModel.makeClient().fetchHistory(
                sessionId: sessionId,
                offset: loadedConversationCount
            )
            let olderMessages = renderConversationPreviews(history: history, sessionId: sessionId)
            let insertIndex = messages.first?.id.hasPrefix("system-prompt:") == true ? 2 : 0

            messages.insert(contentsOf: olderMessages, at: min(insertIndex, messages.count))
            hasMoreMessages = history.hasMore
            loadedConversationCount += history.conversations.count
        } catch {
            await handle(error)
        }
    }

    func renameSession(sessionId: String, name: String) async {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return
        }

        do {
            try await appModel.makeClient().renameSession(sessionId: sessionId, name: trimmed)
            sessions = sessions.map { item in
                guard item.sessionId == sessionId else {
                    return item
                }

                var updated = item
                updated.name = trimmed
                return updated
            }

            if self.session?.sessionId == sessionId {
                self.session?.name = trimmed
            }
        } catch {
            await handle(error)
        }
    }

    func deleteSession(sessionId: String) async {
        do {
            try await appModel.makeClient().deleteSession(sessionId: sessionId)
            sessions.removeAll { $0.sessionId == sessionId }

            if selectedSessionId == sessionId {
                startNewConversation()
            }
        } catch {
            await handle(error)
        }
    }

    func updateSystemPrompt(_ content: String) async {
        guard let sessionId = selectedSessionId ?? session?.sessionId else {
            return
        }

        do {
            try await appModel.makeClient().updateSystemPrompt(sessionId: sessionId, systemPrompt: content)
            session?.systemPrompt = content

            if messages.first?.id.hasPrefix("system-prompt:") == true {
                messages[0].content = content
            } else {
                messages.insert(.systemPrompt(content, sessionId: sessionId, createdAt: session?.createdAt), at: 0)
                messages.insert(.initialConfirmation(sessionId: sessionId, createdAt: session?.createdAt), at: 1)
            }
        } catch {
            await handle(error)
        }
    }

    func incrementReplyCount(for messageId: String) {
        guard let index = messages.firstIndex(where: { $0.id == messageId }) else {
            return
        }

        let current = messages[index].replyCount ?? 0
        messages[index].replyCount = current + 1
    }

    private func handle(_ error: Error) async {
        if let apiError = error as? APIServiceError, apiError.status == 401 || apiError.status == 403 {
            await appModel.signOut()
            return
        }

        errorMessage = appModel.errorMessage(from: error)
    }

    private func renderConversationHistory(session: SessionInfo?, history: ConversationHistoryPage) -> [Message] {
        let sessionId = session?.sessionId ?? ""
        var rendered: [Message] = []

        if let prompt = session?.systemPrompt, !prompt.isEmpty {
            rendered.append(.systemPrompt(prompt, sessionId: sessionId, createdAt: session?.createdAt))
            rendered.append(.initialConfirmation(sessionId: sessionId, createdAt: session?.createdAt))
        }

        rendered.append(contentsOf: renderConversationPreviews(history: history, sessionId: sessionId))
        return rendered
    }

    private func renderConversationPreviews(history: ConversationHistoryPage, sessionId: String) -> [Message] {
        history.conversations.flatMap { conversation in
            [
                Message(
                    id: "\(conversation.conversationId):user",
                    sessionId: sessionId.isEmpty ? conversation.sessionId : sessionId,
                    role: "user",
                    content: conversation.userMessage,
                    replyCount: nil,
                    createdAt: conversation.createdAt
                ),
                Message(
                    id: conversation.conversationId,
                    sessionId: sessionId.isEmpty ? conversation.sessionId : sessionId,
                    role: "assistant",
                    content: conversation.assistantMessage,
                    replyCount: conversation.replyCount,
                    createdAt: conversation.createdAt
                )
            ]
        }
    }
}
