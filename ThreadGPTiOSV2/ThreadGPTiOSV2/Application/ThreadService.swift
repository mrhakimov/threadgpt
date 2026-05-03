import Foundation

final class ThreadService {
    private let repository: ThreadRepository

    init(repository: ThreadRepository) {
        self.repository = repository
    }

    func loadThreadHistory(conversationID: String) async throws -> ThreadHistoryPage {
        try await repository.fetchThreadMessages(
            conversationID: conversationID,
            limit: ThreadGPTConstants.messagePageSize,
            offset: 0
        )
    }

    func loadOlderThreadMessages(conversationID: String, loadedCount: Int) async throws -> ThreadHistoryPage {
        try await repository.fetchThreadMessages(
            conversationID: conversationID,
            limit: ThreadGPTConstants.messagePageSize,
            offset: loadedCount
        )
    }

    func streamThreadTurn(
        conversationID: String,
        userMessage: String
    ) -> AsyncThrowingStream<StreamEvent, Error> {
        repository.sendThreadMessage(conversationID: conversationID, userMessage: userMessage)
    }
}

