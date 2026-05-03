import Foundation

final class SessionService {
    private let repository: SessionRepository

    init(repository: SessionRepository) {
        self.repository = repository
    }

    func listSessions(limit: Int = ThreadGPTConstants.sessionPageSize, offset: Int = 0) async throws -> SessionListPage {
        try await repository.fetchSessions(limit: limit, offset: offset)
    }

    func renameConversation(sessionID: String, name: String) async throws -> String {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, trimmed.count <= 256 else { return "" }
        try await repository.renameSession(sessionID: sessionID, name: trimmed)
        return trimmed
    }

    func deleteConversation(sessionID: String) async throws {
        try await repository.deleteSession(sessionID: sessionID)
    }

    func updateConversationSystemPrompt(sessionID: String, content: String) async throws {
        try await repository.updateSystemPrompt(sessionID: sessionID, systemPrompt: content)
    }
}

