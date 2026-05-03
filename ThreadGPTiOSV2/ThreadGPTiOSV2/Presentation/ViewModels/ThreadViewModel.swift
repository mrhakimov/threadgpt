import Foundation

@MainActor
final class ThreadViewModel: ObservableObject {
    @Published private(set) var messages: [ChatMessage] = []
    @Published private(set) var hasMore = false
    @Published private(set) var isLoading = false
    @Published private(set) var isLoadingMore = false
    @Published private(set) var isSending = false
    @Published var streamingContent = ""
    @Published var errorMessage: String?

    let parentMessage: ChatMessage
    private let service: ThreadService
    private let onReplySent: () -> Void
    private var sendTask: Task<Void, Never>?

    init(parentMessage: ChatMessage, service: ThreadService, onReplySent: @escaping () -> Void) {
        self.parentMessage = parentMessage
        self.service = service
        self.onReplySent = onReplySent
    }

    func load() {
        Task {
            isLoading = true
            do {
                let page = try await service.loadThreadHistory(conversationID: parentMessage.id)
                messages = page.messages
                hasMore = page.hasMore
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }

    func loadMore() {
        guard hasMore, !isLoadingMore else { return }
        Task {
            isLoadingMore = true
            do {
                let page = try await service.loadOlderThreadMessages(conversationID: parentMessage.id, loadedCount: messages.count)
                messages = page.messages + messages
                hasMore = page.hasMore
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoadingMore = false
        }
    }

    func sendMessage(_ content: String) {
        guard !isSending else { return }
        sendTask?.cancel()
        sendTask = Task {
            isSending = true
            errorMessage = nil
            messages.append(createOptimisticUserMessage(content, sessionID: parentMessage.sessionID))
            streamingContent = ""

            var accumulated = ""
            do {
                for try await event in service.streamThreadTurn(conversationID: parentMessage.id, userMessage: content) {
                    if case .chunk(let chunk) = event {
                        accumulated += chunk
                        streamingContent = accumulated
                    }
                }
                messages.append(ChatMessage(
                    id: UUID().uuidString,
                    sessionID: parentMessage.sessionID,
                    role: .assistant,
                    content: accumulated,
                    replyCount: nil,
                    createdAt: ISO8601DateFormatter().string(from: Date())
                ))
                streamingContent = ""
                onReplySent()
            } catch {
                let appError = error.asAppError
                if appError != .cancelled {
                    errorMessage = appError.localizedDescription
                }
                streamingContent = ""
            }
            isSending = false
        }
    }

    func cancel() {
        sendTask?.cancel()
    }
}

