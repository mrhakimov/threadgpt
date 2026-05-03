import Foundation

final class RemoteAuthRepository: AuthRepository {
    private let api = APIClient.shared

    func login(apiKey: String) async throws -> AuthResponse {
        let response: AuthResponse = try await api.request(
            method: "POST",
            path: "/api/auth",
            body: ["api_key": apiKey]
        )
        api.setToken(response.token)
        return response
    }

    func checkAuth() async throws {
        try await api.requestVoid(method: "GET", path: "/api/auth/check")
    }

    func logout() async throws {
        try await api.requestVoid(method: "DELETE", path: "/api/auth/logout")
        api.setToken(nil)
    }

    func fetchAuthInfo() async throws -> AuthInfo {
        try await api.request(method: "GET", path: "/api/auth/info")
    }
}
