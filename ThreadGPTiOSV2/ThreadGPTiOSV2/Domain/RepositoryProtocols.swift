import Foundation

protocol AuthRepository {
    func authenticate(apiKey: String) async throws
    func checkAuthentication() async throws -> Bool
    func fetchAuthInfo() async throws -> AuthInfo
    func logout() async throws
    func hasStoredToken() -> Bool
}

protocol SessionRepository {
    func initSession() async throws -> ChatSession
    func fetchSession(sessionID: String) async throws -> ChatSession
    func fetchSessions(limit: Int, offset: Int) async throws -> SessionListPage
    func createSession(name: String) async throws -> ChatSession
    func renameSession(sessionID: String, name: String) async throws
    func updateSystemPrompt(sessionID: String, systemPrompt: String) async throws
    func deleteSession(sessionID: String) async throws
    func fetchHistory(sessionID: String?, limit: Int, offset: Int) async throws -> ConversationHistoryPage
}

protocol ChatRepository {
    func sendChatMessage(
        userMessage: String,
        sessionID: String?,
        forceNew: Bool
    ) -> AsyncThrowingStream<StreamEvent, Error>
}

protocol ThreadRepository {
    func fetchThreadMessages(conversationID: String, limit: Int, offset: Int) async throws -> ThreadHistoryPage
    func sendThreadMessage(conversationID: String, userMessage: String) -> AsyncThrowingStream<StreamEvent, Error>
}

