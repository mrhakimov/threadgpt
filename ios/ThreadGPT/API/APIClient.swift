import Foundation

struct APIServiceError: LocalizedError, Equatable {
    let message: String
    let code: String?
    let status: Int?

    var errorDescription: String? {
        message
    }

    static let invalidServerURL = APIServiceError(
        message: "The backend URL is not valid.",
        code: "invalid_server_url",
        status: nil
    )
}

@MainActor
struct APIClient {
    let baseURL: URL
    let bearerToken: String?

    private let decoder: JSONDecoder
    private let encoder: JSONEncoder

    init(baseURL: URL, bearerToken: String?) {
        self.baseURL = baseURL
        self.bearerToken = bearerToken

        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        self.decoder = decoder

        let encoder = JSONEncoder()
        encoder.keyEncodingStrategy = .convertToSnakeCase
        self.encoder = encoder
    }

    func authenticate(apiKey: String) async throws -> AuthResponse {
        try await request("api/auth", method: "POST", body: AuthRequest(apiKey: apiKey))
    }

    func checkAuthentication() async -> Bool {
        do {
            try await requestVoid("api/auth/check")
            return true
        } catch {
            return false
        }
    }

    func fetchAuthInfo() async throws -> AuthInfo {
        try await request("api/auth/info")
    }

    func logout() async {
        try? await requestVoid("api/auth/logout", method: "DELETE")
    }

    func initSession() async throws -> SessionInfo {
        try await request("api/session", method: "POST", body: EmptyRequest())
    }

    func fetchSession(sessionId: String) async throws -> SessionInfo {
        try await request("api/sessions/\(sessionId)")
    }

    func fetchSessions(limit: Int = AppConfig.sessionPageSize, offset: Int = 0) async throws -> SessionListResponse {
        try await request(
            "api/sessions",
            queryItems: [
                URLQueryItem(name: "limit", value: "\(limit)"),
                URLQueryItem(name: "offset", value: "\(offset)")
            ]
        )
    }

    func createSession(name: String) async throws -> SessionInfo {
        try await request("api/sessions", method: "POST", body: CreateSessionRequest(name: name))
    }

    func renameSession(sessionId: String, name: String) async throws {
        try await requestVoid(
            "api/sessions/\(sessionId)",
            method: "PATCH",
            body: UpdateSessionRequest(name: name, systemPrompt: nil)
        )
    }

    func updateSystemPrompt(sessionId: String, systemPrompt: String) async throws {
        try await requestVoid(
            "api/sessions/\(sessionId)",
            method: "PATCH",
            body: UpdateSessionRequest(name: nil, systemPrompt: systemPrompt)
        )
    }

    func deleteSession(sessionId: String) async throws {
        try await requestVoid("api/sessions/\(sessionId)", method: "DELETE")
    }

    func fetchHistory(sessionId: String?, limit: Int = AppConfig.messagePageSize, offset: Int = 0) async throws -> ConversationHistoryPage {
        var headers: [String: String] = [:]
        if let sessionId {
            headers["X-Session-ID"] = sessionId
        }

        return try await request(
            "api/history",
            queryItems: [
                URLQueryItem(name: "limit", value: "\(limit)"),
                URLQueryItem(name: "offset", value: "\(offset)")
            ],
            headers: headers
        )
    }

    func sendChatMessage(
        _ userMessage: String,
        sessionId: String?,
        forceNew: Bool,
        onChunk: @escaping (String) -> Void
    ) async throws -> String? {
        var request = try makeRequest(path: "api/chat", method: "POST")
        try setJSONBody(
            ChatRequest(
                userMessage: userMessage,
                sessionId: sessionId ?? "",
                forceNew: forceNew
            ),
            on: &request
        )

        return try await stream(request: request, onChunk: onChunk)
    }

    func fetchThreadMessages(conversationId: String, limit: Int = AppConfig.messagePageSize, offset: Int = 0) async throws -> MessageHistoryPage {
        try await request(
            "api/thread",
            queryItems: [
                URLQueryItem(name: "conversation_id", value: conversationId),
                URLQueryItem(name: "limit", value: "\(limit)"),
                URLQueryItem(name: "offset", value: "\(offset)")
            ]
        )
    }

    func sendThreadMessage(
        conversationId: String,
        userMessage: String,
        onChunk: @escaping (String) -> Void
    ) async throws {
        var request = try makeRequest(path: "api/thread", method: "POST")
        try setJSONBody(
            ThreadRequest(conversationId: conversationId, userMessage: userMessage),
            on: &request
        )

        _ = try await stream(request: request, onChunk: onChunk)
    }

    private func request<T: Decodable>(
        _ path: String,
        method: String = "GET",
        queryItems: [URLQueryItem] = [],
        headers: [String: String] = [:]
    ) async throws -> T {
        let request = try makeRequest(path: path, method: method, queryItems: queryItems, headers: headers)
        return try await perform(request)
    }

    private func request<T: Decodable, Body: Encodable>(
        _ path: String,
        method: String,
        queryItems: [URLQueryItem] = [],
        headers: [String: String] = [:],
        body: Body
    ) async throws -> T {
        var request = try makeRequest(path: path, method: method, queryItems: queryItems, headers: headers)
        try setJSONBody(body, on: &request)
        return try await perform(request)
    }

    private func requestVoid(
        _ path: String,
        method: String = "GET",
        queryItems: [URLQueryItem] = [],
        headers: [String: String] = [:]
    ) async throws {
        let request = try makeRequest(path: path, method: method, queryItems: queryItems, headers: headers)
        try await performVoid(request)
    }

    private func requestVoid<Body: Encodable>(
        _ path: String,
        method: String,
        queryItems: [URLQueryItem] = [],
        headers: [String: String] = [:],
        body: Body
    ) async throws {
        var request = try makeRequest(path: path, method: method, queryItems: queryItems, headers: headers)
        try setJSONBody(body, on: &request)
        try await performVoid(request)
    }

    private func makeRequest(
        path: String,
        method: String,
        queryItems: [URLQueryItem] = [],
        headers: [String: String] = [:]
    ) throws -> URLRequest {
        let trimmedPath = path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        var components = URLComponents(
            url: baseURL.appendingPathComponent(trimmedPath),
            resolvingAgainstBaseURL: false
        )
        components?.queryItems = queryItems.isEmpty ? nil : queryItems

        guard let url = components?.url else {
            throw APIServiceError.invalidServerURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Accept")

        if let bearerToken, !bearerToken.isEmpty {
            request.setValue("Bearer \(bearerToken)", forHTTPHeaderField: "Authorization")
        }

        headers.forEach { key, value in
            request.setValue(value, forHTTPHeaderField: key)
        }

        return request
    }

    private func setJSONBody<Body: Encodable>(_ body: Body, on request: inout URLRequest) throws {
        request.httpBody = try encoder.encode(body)
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    }

    private func perform<T: Decodable>(_ request: URLRequest) async throws -> T {
        let (data, response) = try await URLSession.shared.data(for: request)
        try validate(response: response, data: data)
        return try decoder.decode(T.self, from: data)
    }

    private func performVoid(_ request: URLRequest) async throws {
        let (data, response) = try await URLSession.shared.data(for: request)
        try validate(response: response, data: data)
    }

    private func stream(request: URLRequest, onChunk: @escaping (String) -> Void) async throws -> String? {
        let (bytes, response) = try await URLSession.shared.bytes(for: request)
        try validate(response: response, data: Data())

        var resolvedSessionId: String?
        for try await line in bytes.lines {
            guard line.hasPrefix("data: ") else {
                continue
            }

            let payload = line.dropFirst(6).trimmingCharacters(in: .whitespacesAndNewlines)
            if payload == "[DONE]" {
                return resolvedSessionId
            }

            guard let data = payload.data(using: .utf8) else {
                continue
            }

            if let errorEnvelope = try? decoder.decode(APIErrorEnvelope.self, from: data) {
                throw APIServiceError(
                    message: errorEnvelope.error.message,
                    code: errorEnvelope.error.code,
                    status: errorEnvelope.error.status
                )
            }

            guard let event = try? decoder.decode(StreamEvent.self, from: data) else {
                continue
            }

            if let sessionId = event.sessionId, !sessionId.isEmpty {
                resolvedSessionId = sessionId
            }

            if let chunk = event.chunk, !chunk.isEmpty {
                onChunk(chunk)
            }
        }

        return resolvedSessionId
    }

    private func validate(response: URLResponse, data: Data) throws {
        guard let http = response as? HTTPURLResponse else {
            return
        }

        guard (200..<300).contains(http.statusCode) else {
            if let errorEnvelope = try? decoder.decode(APIErrorEnvelope.self, from: data) {
                throw APIServiceError(
                    message: errorEnvelope.error.message,
                    code: errorEnvelope.error.code,
                    status: errorEnvelope.error.status ?? http.statusCode
                )
            }

            let fallback = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
            throw APIServiceError(
                message: fallback?.isEmpty == false ? fallback! : defaultErrorMessage(statusCode: http.statusCode),
                code: nil,
                status: http.statusCode
            )
        }
    }

    private func defaultErrorMessage(statusCode: Int) -> String {
        if statusCode >= 500 {
            return "Something went wrong on the server. Please try again."
        }
        if statusCode == 401 || statusCode == 403 {
            return "Please sign in again."
        }
        if statusCode == 404 {
            return "That resource was not found."
        }
        if statusCode == 429 {
            return "Too many requests. Please wait a moment and try again."
        }
        return "Something went wrong."
    }
}
