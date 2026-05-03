import Foundation

@MainActor
final class ThreadViewModel: ObservableObject {
    @Published var messages: [Message] = []
    @Published var streamingContent = ""
    @Published var errorMessage: String?
    @Published var isLoading = false
    @Published var isSending = false
    @Published var hasMoreMessages = false

    let parentMessage: Message

    private let appModel: AppViewModel
    private var loadedMessageCount = 0

    init(parentMessage: Message, appModel: AppViewModel) {
        self.parentMessage = parentMessage
        self.appModel = appModel
    }

    func load() async {
        isLoading = true
        errorMessage = nil

        do {
            let page = try await appModel.makeClient().fetchThreadMessages(conversationId: parentMessage.id)
            messages = page.messages
            hasMoreMessages = page.hasMore
            loadedMessageCount = page.messages.count
        } catch {
            await handle(error)
        }

        isLoading = false
    }

    func sendMessage(_ content: String, onReplySent: @escaping () -> Void) async {
        let trimmed = content.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, !isSending else {
            return
        }

        isSending = true
        errorMessage = nil
        streamingContent = ""
        messages.append(.optimisticUser(trimmed, sessionId: parentMessage.sessionId))

        do {
            var accumulated = ""
            let client = try appModel.makeClient()
            try await client.sendThreadMessage(
                conversationId: parentMessage.id,
                userMessage: trimmed
            ) { chunk in
                accumulated += chunk
                self.streamingContent = accumulated
            }

            let page = try await client.fetchThreadMessages(conversationId: parentMessage.id)
            messages = page.messages
            hasMoreMessages = page.hasMore
            loadedMessageCount = page.messages.count
            streamingContent = ""
            onReplySent()
        } catch {
            streamingContent = ""
            await handle(error)
        }

        isSending = false
    }

    func loadMoreMessages() async {
        guard hasMoreMessages else {
            return
        }

        do {
            let page = try await appModel.makeClient().fetchThreadMessages(
                conversationId: parentMessage.id,
                offset: loadedMessageCount
            )
            messages.insert(contentsOf: page.messages, at: 0)
            hasMoreMessages = page.hasMore
            loadedMessageCount += page.messages.count
        } catch {
            await handle(error)
        }
    }

    private func handle(_ error: Error) async {
        if let apiError = error as? APIServiceError, apiError.status == 401 || apiError.status == 403 {
            await appModel.signOut()
            return
        }

        errorMessage = appModel.errorMessage(from: error)
    }
}
