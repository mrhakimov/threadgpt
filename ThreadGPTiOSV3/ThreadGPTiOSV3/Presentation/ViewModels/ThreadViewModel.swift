import Foundation

@MainActor
final class ThreadViewModel: ObservableObject {
    @Published var messages: [Message] = []
    @Published var streamingContent = ""
    @Published var isStreaming = false
    @Published var isSending = false
    @Published var isLoading = false
    @Published var hasLoadedMessages = false
    @Published var isLoadingSystemPrompt = false
    @Published var hasMore = false
    @Published var error: String?
    @Published var systemPrompt: String

    let parentMessage: Message
    var onReplySent: (() -> Void)?

    private let threadRepo: ThreadRepository
    private let sessionRepo: SessionRepository

    init(
        parentMessage: Message,
        systemPrompt: String? = nil,
        threadRepo: ThreadRepository = AppContainer.shared.threadRepo,
        sessionRepo: SessionRepository = AppContainer.shared.sessionRepo
    ) {
        self.parentMessage = parentMessage
        self.systemPrompt = systemPrompt ?? ""
        self.threadRepo = threadRepo
        self.sessionRepo = sessionRepo
    }

    var conversationId: String { parentMessage.id }

    func loadSystemPrompt() async {
        isLoadingSystemPrompt = true
        defer { isLoadingSystemPrompt = false }

        do {
            let session = try await sessionRepo.getSession(id: parentMessage.sessionId)
            systemPrompt = session.systemPrompt ?? ""
        } catch {
            self.error = error.localizedDescription
        }
    }

    func loadMessages() async {
        isLoading = true
        hasLoadedMessages = false
        defer {
            isLoading = false
            hasLoadedMessages = true
        }

        do {
            let page = try await threadRepo.fetchThread(
                conversationId: conversationId, limit: 50, offset: 0
            )
            messages = page.messages
            hasMore = page.hasMore
        } catch {
            self.error = error.localizedDescription
        }
    }

    func loadMore() async {
        guard hasMore, !isLoading else { return }
        isLoading = true
        do {
            let page = try await threadRepo.fetchThread(
                conversationId: conversationId, limit: 50, offset: messages.count
            )
            messages.insert(contentsOf: page.messages, at: 0)
            hasMore = page.hasMore
        } catch {}
        isLoading = false
    }

    func sendReply(_ content: String) async {
        let trimmed = content.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }

        error = nil
        isSending = true
        isStreaming = true
        streamingContent = ""

        // Optimistic user message
        let userMsg = Message(
            id: UUID().uuidString,
            sessionId: parentMessage.sessionId,
            role: .user,
            content: trimmed,
            replyCount: 0,
            createdAt: ISO8601DateFormatter().string(from: Date())
        )
        messages.append(userMsg)

        do {
            var accumulated = ""
            try await threadRepo.sendReply(
                conversationId: conversationId,
                userMessage: trimmed
            ) { [weak self] chunk in
                accumulated += chunk
                Task { @MainActor in
                    self?.streamingContent = accumulated
                }
            }

            // Add assistant message
            let assistantMsg = Message(
                id: UUID().uuidString,
                sessionId: parentMessage.sessionId,
                role: .assistant,
                content: accumulated,
                replyCount: 0,
                createdAt: ISO8601DateFormatter().string(from: Date())
            )
            messages.append(assistantMsg)
            streamingContent = ""
            isStreaming = false
            isSending = false
            onReplySent?()
        } catch {
            self.error = error.localizedDescription
            streamingContent = ""
            isStreaming = false
            isSending = false
        }
    }

}
