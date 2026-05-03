import Foundation

final class APIClient {
    private let preferences: AppPreferences
    private let tokenStore: TokenStore
    private let session: URLSession
    private let decoder = JSONDecoder()
    private let encoder = JSONEncoder()

    init(
        preferences: AppPreferences,
        tokenStore: TokenStore,
        session: URLSession = .shared
    ) {
        self.preferences = preferences
        self.tokenStore = tokenStore
        self.session = session
    }

    func encodedBody<T: Encodable>(_ value: T) throws -> Data {
        try encoder.encode(value)
    }

    func request<T: Decodable>(
        _ path: String,
        method: String = "GET",
        queryItems: [URLQueryItem] = [],
        body: Data? = nil,
        headers: [String: String] = [:],
        authenticated: Bool = true
    ) async throws -> T {
        let request = try makeRequest(
            path,
            method: method,
            queryItems: queryItems,
            body: body,
            headers: headers,
            authenticated: authenticated
        )
        let (data, response) = try await session.data(for: request)
        try validate(response: response, data: data)
        return try decoder.decode(T.self, from: data)
    }

    func requestVoid(
        _ path: String,
        method: String = "GET",
        queryItems: [URLQueryItem] = [],
        body: Data? = nil,
        headers: [String: String] = [:],
        authenticated: Bool = true
    ) async throws {
        let request = try makeRequest(
            path,
            method: method,
            queryItems: queryItems,
            body: body,
            headers: headers,
            authenticated: authenticated
        )
        let (data, response) = try await session.data(for: request)
        try validate(response: response, data: data)
    }

    func status(
        _ path: String,
        method: String = "GET",
        authenticated: Bool = true
    ) async throws -> Int {
        let request = try makeRequest(path, method: method, authenticated: authenticated)
        let (_, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw AppError.invalidResponse
        }
        return httpResponse.statusCode
    }

    func stream(
        _ path: String,
        method: String = "POST",
        queryItems: [URLQueryItem] = [],
        body: Data? = nil,
        authenticated: Bool = true
    ) -> AsyncThrowingStream<StreamEvent, Error> {
        AsyncThrowingStream { continuation in
            let task = Task {
                do {
                    var request = try makeRequest(
                        path,
                        method: method,
                        queryItems: queryItems,
                        body: body,
                        headers: ["Accept": "text/event-stream"],
                        authenticated: authenticated
                    )
                    request.timeoutInterval = 300

                    let (bytes, response) = try await session.bytes(for: request)
                    guard let httpResponse = response as? HTTPURLResponse else {
                        throw AppError.invalidResponse
                    }
                    guard (200..<300).contains(httpResponse.statusCode) else {
                        let text = try await collectText(from: bytes)
                        throw error(status: httpResponse.statusCode, data: Data(text.utf8))
                    }

                    for try await line in bytes.lines {
                        guard line.hasPrefix("data: ") else { continue }
                        let data = String(line.dropFirst(6)).trimmingCharacters(in: .whitespacesAndNewlines)
                        if data == "[DONE]" {
                            continuation.finish()
                            return
                        }

                        guard let payloadData = data.data(using: .utf8) else { continue }
                        if let envelope = try? decoder.decode(APIErrorEnvelope.self, from: payloadData) {
                            throw AppError.api(envelope.error)
                        }
                        if let payload = try? decoder.decode(StreamPayload.self, from: payloadData) {
                            if let sessionID = payload.sessionID, !sessionID.isEmpty {
                                continuation.yield(.sessionID(sessionID))
                            }
                            if let chunk = payload.chunk, !chunk.isEmpty {
                                continuation.yield(.chunk(chunk))
                            }
                        }
                    }

                    continuation.finish()
                } catch {
                    if Task.isCancelled {
                        continuation.finish(throwing: AppError.cancelled)
                    } else {
                        continuation.finish(throwing: error.asAppError)
                    }
                }
            }

            continuation.onTermination = { _ in
                task.cancel()
            }
        }
    }

    private func makeRequest(
        _ path: String,
        method: String = "GET",
        queryItems: [URLQueryItem] = [],
        body: Data? = nil,
        headers: [String: String] = [:],
        authenticated: Bool = true
    ) throws -> URLRequest {
        let trimmedBase = preferences.serverURL.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        guard var components = URLComponents(string: trimmedBase + path) else {
            throw AppError.invalidURL
        }
        if !queryItems.isEmpty {
            components.queryItems = queryItems
        }
        guard let url = components.url else {
            throw AppError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = method
        request.httpBody = body
        request.cachePolicy = .reloadIgnoringLocalCacheData
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        if body != nil {
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        if authenticated, let token = tokenStore.readToken() {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        headers.forEach { request.setValue($0.value, forHTTPHeaderField: $0.key) }
        return request
    }

    private func validate(response: URLResponse, data: Data) throws {
        guard let httpResponse = response as? HTTPURLResponse else {
            throw AppError.invalidResponse
        }
        guard (200..<300).contains(httpResponse.statusCode) else {
            throw error(status: httpResponse.statusCode, data: data)
        }
    }

    private func error(status: Int, data: Data) -> AppError {
        if let envelope = try? decoder.decode(APIErrorEnvelope.self, from: data) {
            return .api(envelope.error)
        }
        if let text = String(data: data, encoding: .utf8), !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return .http(status: status, message: text)
        }
        if status == 401 || status == 403 {
            return .unauthorized
        }
        if status == 429 {
            return .http(status: status, message: "Too many requests. Please wait a moment and try again.")
        }
        if status >= 500 {
            return .http(status: status, message: "Something went wrong on the server. Please try again.")
        }
        return .http(status: status, message: "Something went wrong.")
    }

    private func collectText(from bytes: URLSession.AsyncBytes) async throws -> String {
        var output = ""
        for try await line in bytes.lines {
            output += line
        }
        return output
    }
}

private struct StreamPayload: Decodable {
    let sessionID: String?
    let chunk: String?

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case chunk
    }
}

