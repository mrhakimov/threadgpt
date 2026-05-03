import Foundation

final class RemoteAuthRepository: AuthRepository {
    private let api: APIClient
    private let tokenStore: TokenStore

    init(api: APIClient, tokenStore: TokenStore) {
        self.api = api
        self.tokenStore = tokenStore
    }

    func authenticate(apiKey: String) async throws {
        struct Request: Encodable { let apiKey: String; enum CodingKeys: String, CodingKey { case apiKey = "api_key" } }
        struct Response: Decodable { let token: String? }

        let body = try api.encodedBody(Request(apiKey: apiKey))
        let response: Response = try await api.request("/api/auth", method: "POST", body: body, authenticated: false)
        guard let token = response.token, !token.isEmpty else {
            throw AppError.invalidResponse
        }
        try tokenStore.saveToken(token)
    }

    func checkAuthentication() async throws -> Bool {
        guard hasStoredToken() else { return false }
        let status = try await api.status("/api/auth/check")
        if status == 401 || status == 403 {
            return false
        }
        if !(200..<300).contains(status) {
            throw AppError.http(status: status, message: "Unable to verify the current session.")
        }
        return (200..<300).contains(status)
    }

    func fetchAuthInfo() async throws -> AuthInfo {
        try await api.request("/api/auth/info")
    }

    func logout() async throws {
        defer { tokenStore.clearToken() }
        try await api.requestVoid("/api/auth/logout", method: "DELETE")
    }

    func hasStoredToken() -> Bool {
        tokenStore.readToken()?.isEmpty == false
    }
}

final class RemoteSessionRepository: SessionRepository {
    private let api: APIClient

    init(api: APIClient) {
        self.api = api
    }

    func initSession() async throws -> ChatSession {
        try await api.request("/api/session", method: "POST", body: Data("{}".utf8))
    }

    func fetchSession(sessionID: String) async throws -> ChatSession {
        try await api.request("/api/sessions/\(sessionID)")
    }

    func fetchSessions(limit: Int, offset: Int) async throws -> SessionListPage {
        try await api.request(
            "/api/sessions",
            queryItems: [
                URLQueryItem(name: "limit", value: String(limit)),
                URLQueryItem(name: "offset", value: String(offset))
            ]
        )
    }

    func createSession(name: String) async throws -> ChatSession {
        struct Request: Encodable { let name: String }
        let body = try api.encodedBody(Request(name: name))
        return try await api.request("/api/sessions", method: "POST", body: body)
    }

    func renameSession(sessionID: String, name: String) async throws {
        struct Request: Encodable { let name: String }
        let body = try api.encodedBody(Request(name: name))
        try await api.requestVoid("/api/sessions/\(sessionID)", method: "PATCH", body: body)
    }

    func updateSystemPrompt(sessionID: String, systemPrompt: String) async throws {
        struct Request: Encodable {
            let systemPrompt: String
            enum CodingKeys: String, CodingKey { case systemPrompt = "system_prompt" }
        }
        let body = try api.encodedBody(Request(systemPrompt: systemPrompt))
        try await api.requestVoid("/api/sessions/\(sessionID)", method: "PATCH", body: body)
    }

    func deleteSession(sessionID: String) async throws {
        try await api.requestVoid("/api/sessions/\(sessionID)", method: "DELETE")
    }

    func fetchHistory(sessionID: String?, limit: Int, offset: Int) async throws -> ConversationHistoryPage {
        var headers: [String: String] = [:]
        if let sessionID, !sessionID.isEmpty {
            headers["X-Session-ID"] = sessionID
        }
        return try await api.request(
            "/api/history",
            queryItems: [
                URLQueryItem(name: "limit", value: String(limit)),
                URLQueryItem(name: "offset", value: String(offset))
            ],
            headers: headers
        )
    }
}

final class RemoteChatRepository: ChatRepository {
    private let api: APIClient

    init(api: APIClient) {
        self.api = api
    }

    func sendChatMessage(
        userMessage: String,
        sessionID: String?,
        forceNew: Bool
    ) -> AsyncThrowingStream<StreamEvent, Error> {
        struct Request: Encodable {
            let userMessage: String
            let sessionID: String
            let forceNew: Bool

            enum CodingKeys: String, CodingKey {
                case userMessage = "user_message"
                case sessionID = "session_id"
                case forceNew = "force_new"
            }
        }

        do {
            let body = try api.encodedBody(Request(
                userMessage: userMessage,
                sessionID: sessionID ?? "",
                forceNew: forceNew
            ))
            return api.stream("/api/chat", method: "POST", body: body)
        } catch {
            return AsyncThrowingStream { continuation in
                continuation.finish(throwing: error)
            }
        }
    }
}

final class RemoteThreadRepository: ThreadRepository {
    private let api: APIClient

    init(api: APIClient) {
        self.api = api
    }

    func fetchThreadMessages(conversationID: String, limit: Int, offset: Int) async throws -> ThreadHistoryPage {
        try await api.request(
            "/api/thread",
            queryItems: [
                URLQueryItem(name: "conversation_id", value: conversationID),
                URLQueryItem(name: "limit", value: String(limit)),
                URLQueryItem(name: "offset", value: String(offset))
            ]
        )
    }

    func sendThreadMessage(conversationID: String, userMessage: String) -> AsyncThrowingStream<StreamEvent, Error> {
        struct Request: Encodable {
            let conversationID: String
            let userMessage: String

            enum CodingKeys: String, CodingKey {
                case conversationID = "conversation_id"
                case userMessage = "user_message"
            }
        }

        do {
            let body = try api.encodedBody(Request(conversationID: conversationID, userMessage: userMessage))
            return api.stream("/api/thread", method: "POST", body: body)
        } catch {
            return AsyncThrowingStream { continuation in
                continuation.finish(throwing: error)
            }
        }
    }
}
