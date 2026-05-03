import Foundation

final class APIClient {
    static let shared = APIClient()

    #if targetEnvironment(simulator)
    private let baseURL = "http://localhost:8000"
    #else
    private let baseURL = "http://192.168.1.139:8000"
    #endif

    private let session: URLSession
    private var token: String?

    private init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        self.session = URLSession(configuration: config)
        self.token = TokenStore.loadToken()
    }

    func setToken(_ token: String?) {
        self.token = token
        if let token {
            TokenStore.saveToken(token)
        } else {
            TokenStore.clearToken()
        }
    }

    var hasToken: Bool { token != nil }

    // MARK: - Standard Request

    func request<T: Decodable>(
        method: String,
        path: String,
        body: Encodable? = nil,
        headers: [String: String] = [:],
        queryItems: [URLQueryItem]? = nil
    ) async throws -> T {
        let data = try await rawRequest(method: method, path: path, body: body, headers: headers, queryItems: queryItems)
        do {
            return try JSONDecoder().decode(T.self, from: data)
        } catch {
            throw AppError.decodingError(error.localizedDescription)
        }
    }

    func requestVoid(
        method: String,
        path: String,
        body: Encodable? = nil,
        headers: [String: String] = [:],
        queryItems: [URLQueryItem]? = nil
    ) async throws {
        _ = try await rawRequest(method: method, path: path, body: body, headers: headers, queryItems: queryItems)
    }

    // MARK: - SSE Stream

    func stream(
        method: String,
        path: String,
        body: Encodable? = nil,
        onChunk: @escaping (String) -> Void
    ) async throws -> String? {
        let request = try buildRequest(method: method, path: path, body: body, queryItems: nil, headers: [:])

        let (bytes, response) = try await session.bytes(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw AppError.networkError("Invalid response")
        }

        if httpResponse.statusCode != 200 {
            var collected = Data()
            for try await byte in bytes {
                collected.append(byte)
                if collected.count > 4096 { break }
            }
            throw AppError.from(statusCode: httpResponse.statusCode, data: collected)
        }

        var resolvedSessionId: String?
        var buffer = ""

        for try await byte in bytes {
            buffer.append(Character(UnicodeScalar(byte)))
            guard buffer.hasSuffix("\n") else { continue }

            let line = buffer.trimmingCharacters(in: .whitespacesAndNewlines)
            buffer = ""

            guard line.hasPrefix("data: ") else { continue }
            let payload = String(line.dropFirst(6))

            if payload == "[DONE]" { break }

            guard let jsonData = payload.data(using: .utf8) else { continue }

            // Check for error
            if let errorWrapper = try? JSONDecoder().decode(SSEError.self, from: jsonData) {
                throw AppError.from(payload: errorWrapper.error)
            }

            if let chunk = try? JSONDecoder().decode(SSEChunk.self, from: jsonData) {
                if let sid = chunk.sessionId {
                    resolvedSessionId = sid
                }
                if let text = chunk.chunk {
                    onChunk(text)
                }
            }
        }

        return resolvedSessionId
    }

    // MARK: - Private

    private func rawRequest(
        method: String,
        path: String,
        body: Encodable? = nil,
        headers: [String: String],
        queryItems: [URLQueryItem]?
    ) async throws -> Data {
        let request = try buildRequest(method: method, path: path, body: body, queryItems: queryItems, headers: headers)
        let (data, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw AppError.networkError("Invalid response")
        }

        guard (200...299).contains(httpResponse.statusCode) else {
            throw AppError.from(statusCode: httpResponse.statusCode, data: data)
        }

        return data
    }

    private func buildRequest(
        method: String,
        path: String,
        body: Encodable?,
        queryItems: [URLQueryItem]?,
        headers: [String: String]
    ) throws -> URLRequest {
        var components = URLComponents(string: baseURL + path)!
        if let queryItems, !queryItems.isEmpty {
            components.queryItems = queryItems
        }

        var request = URLRequest(url: components.url!)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if let token {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        for (key, value) in headers {
            request.setValue(value, forHTTPHeaderField: key)
        }

        if let body {
            request.httpBody = try JSONEncoder().encode(body)
        }

        return request
    }
}

// MARK: - SSE Types

private struct SSEChunk: Codable {
    let chunk: String?
    let sessionId: String?

    enum CodingKeys: String, CodingKey {
        case chunk
        case sessionId = "session_id"
    }
}

private struct SSEError: Codable {
    let error: APIErrorPayload
}
