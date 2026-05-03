import Foundation

@MainActor
final class ChatViewModel: ObservableObject {
    @Published private(set) var messages: [ChatMessage] = []
    @Published private(set) var hasMoreMessages = false
    @Published private(set) var loadedConversationCount = 0
    @Published private(set) var session: ChatSession?
    @Published private(set) var isLoading = false
    @Published private(set) var isLoadingMore = false
    @Published private(set) var isSending = false
    @Published var streamingContent = ""
    @Published var errorMessage: String?

    var onSessionResolved: ((String) -> Void)?
    var onUnauthorized: (() -> Void)?
    var onConversationMutated: (() -> Void)?

    private let chatService: ChatService
    private let sessionService: SessionService
    private var selectedSessionID: String?
    private var loadTask: Task<Void, Never>?
    private var sendTask: Task<Void, Never>?

    init(chatService: ChatService, sessionService: SessionService) {
        self.chatService = chatService
        self.sessionService = sessionService
    }

    var isEmpty: Bool {
        messages.isEmpty && streamingContent.isEmpty
    }

    var isFirstMessage: Bool {
        !isLoading && isFirstMessageSession(session)
    }

    var subtitle: String? {
        if let name = session?.name, !name.isEmpty, name != "New conversation" {
            return name
        }
        return session?.systemPrompt
    }

    var composerPlaceholder: String {
        chatInputPlaceholder(isFirstMessage: isFirstMessage)
    }

    func selectSession(_ sessionID: String?) {
        let shouldReload =
            selectedSessionID != sessionID ||
            (sessionID == nil && session != nil) ||
            (sessionID != nil && session?.sessionID != sessionID)
        guard shouldReload else { return }
        selectedSessionID = sessionID
        sendTask?.cancel()
        loadTask?.cancel()
        messages = []
        session = nil
        hasMoreMessages = false
        loadedConversationCount = 0
        streamingContent = ""
        errorMessage = nil

        guard sessionID != nil else {
            isLoading = false
            return
        }

        loadTask = Task {
            isLoading = true
            do {
                let result = try await chatService.loadChatSession(sessionID: sessionID)
                apply(snapshot: result.snapshot)
            } catch {
                handle(error)
            }
            isLoading = false
        }
    }

    func sendMessage(_ content: String) {
        guard !isSending else { return }
        sendTask?.cancel()
        sendTask = Task {
            isSending = true
            errorMessage = nil

            if !isFirstMessage {
                messages.append(createOptimisticUserMessage(content, sessionID: session?.sessionID))
            }
            streamingContent = ""

            do {
                let result = try await chatService.sendChatTurn(
                    content: content,
                    requestedSessionID: selectedSessionID,
                    currentSession: session,
                    onChunk: { [weak self] accumulated in
                        self?.streamingContent = accumulated
                    }
                )
                apply(snapshot: result.snapshot)
                streamingContent = ""
                if let resolved = result.resolvedSessionID,
                   resolved != selectedSessionID {
                    selectedSessionID = resolved
                    onSessionResolved?(resolved)
                }
                onConversationMutated?()
            } catch {
                handle(error)
                streamingContent = ""
            }
            isSending = false
        }
    }

    func loadMoreMessages() {
        guard !isLoadingMore, hasMoreMessages, let session else { return }
        Task {
            isLoadingMore = true
            do {
                let page = try await chatService.loadOlderMessages(session: session, loadedConversationCount: loadedConversationCount)
                messages = page.messages + messages
                hasMoreMessages = page.hasMore
                loadedConversationCount += page.loadedConversationCount
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoadingMore = false
        }
    }

    func loadAllMessages() {
        guard let session else { return }
        Task {
            isLoadingMore = true
            do {
                let page = try await chatService.loadCompleteHistory(session: session)
                messages = page.messages
                hasMoreMessages = false
                loadedConversationCount = page.loadedConversationCount
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoadingMore = false
        }
    }

    func updateSystemPrompt(_ content: String) async throws {
        guard let sessionID = session?.sessionID else { return }
        try await sessionService.updateConversationSystemPrompt(sessionID: sessionID, content: content)
        session = session.map {
            ChatSession(
                sessionID: $0.sessionID,
                assistantID: $0.assistantID,
                systemPrompt: content,
                name: $0.name,
                isNew: $0.isNew,
                createdAt: $0.createdAt
            )
        }
        if messages.first?.isSystemPrompt == true {
            messages[0].content = content
        }
    }

    func incrementReplyCount(for messageID: String, by amount: Int) {
        messages = messages.map { message in
            guard message.id == messageID else { return message }
            var updated = message
            updated.replyCount = (updated.replyCount ?? 0) + amount
            return updated
        }
    }

    func updateLocalSessionName(sessionID: String, name: String) {
        guard session?.sessionID == sessionID else { return }
        session = session.map {
            ChatSession(
                sessionID: $0.sessionID,
                assistantID: $0.assistantID,
                systemPrompt: $0.systemPrompt,
                name: name,
                isNew: $0.isNew,
                createdAt: $0.createdAt
            )
        }
    }

    private func apply(snapshot: ChatStateSnapshot) {
        messages = snapshot.messages
        hasMoreMessages = snapshot.hasMoreMessages
        session = snapshot.session
        loadedConversationCount = snapshot.loadedConversationCount
    }

    private func handle(_ error: Error) {
        let appError = error.asAppError
        if appError.isUnauthorized {
            onUnauthorized?()
        } else if appError != .cancelled {
            errorMessage = appError.localizedDescription
        }
    }
}
