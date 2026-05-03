import Foundation

protocol AuthRepository {
    func login(apiKey: String) async throws -> AuthResponse
    func checkAuth() async throws
    func logout() async throws
    func fetchAuthInfo() async throws -> AuthInfo
}

protocol SessionRepository {
    func createSession() async throws -> Session
    func getSession(id: String) async throws -> Session
    func listSessions(limit: Int, offset: Int) async throws -> SessionsPage
    func createNamedSession(name: String) async throws -> Session
    func updateSession(id: String, name: String?, systemPrompt: String?) async throws
    func deleteSession(id: String) async throws
}

protocol ChatRepository {
    func sendMessage(
        userMessage: String,
        sessionId: String?,
        forceNew: Bool,
        onChunk: @escaping (String) -> Void
    ) async throws -> String?

    func fetchHistory(
        sessionId: String?,
        limit: Int,
        offset: Int
    ) async throws -> ConversationHistoryPage
}

protocol ThreadRepository {
    func fetchThread(
        conversationId: String,
        limit: Int,
        offset: Int
    ) async throws -> ThreadPage

    func sendReply(
        conversationId: String,
        userMessage: String,
        onChunk: @escaping (String) -> Void
    ) async throws
}
