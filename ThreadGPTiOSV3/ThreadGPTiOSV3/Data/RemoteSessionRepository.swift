import Foundation

final class RemoteSessionRepository: SessionRepository {
    private let api = APIClient.shared

    func createSession() async throws -> Session {
        try await api.request(method: "POST", path: "/api/session", body: Empty())
    }

    func getSession(id: String) async throws -> Session {
        try await api.request(method: "GET", path: "/api/sessions/\(id)")
    }

    func listSessions(limit: Int, offset: Int) async throws -> SessionsPage {
        try await api.request(
            method: "GET",
            path: "/api/sessions",
            queryItems: [
                URLQueryItem(name: "limit", value: "\(limit)"),
                URLQueryItem(name: "offset", value: "\(offset)"),
            ]
        )
    }

    func createNamedSession(name: String) async throws -> Session {
        try await api.request(
            method: "POST",
            path: "/api/sessions",
            body: ["name": name]
        )
    }

    func updateSession(id: String, name: String?, systemPrompt: String?) async throws {
        var body: [String: String] = [:]
        if let name { body["name"] = name }
        if let systemPrompt { body["system_prompt"] = systemPrompt }
        try await api.requestVoid(method: "PATCH", path: "/api/sessions/\(id)", body: body)
    }

    func deleteSession(id: String) async throws {
        try await api.requestVoid(method: "DELETE", path: "/api/sessions/\(id)")
    }
}

private struct Empty: Encodable {}
