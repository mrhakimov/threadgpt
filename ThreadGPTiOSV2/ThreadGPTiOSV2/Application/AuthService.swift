import Foundation

final class AuthService {
    private let repository: AuthRepository
    private let preferences: AppPreferences

    init(repository: AuthRepository, preferences: AppPreferences) {
        self.repository = repository
        self.preferences = preferences
    }

    func authenticate(apiKey: String) async throws {
        try await repository.authenticate(apiKey: apiKey)
        preferences.authPersisted = true
    }

    func checkAuthorization() async -> Bool {
        let hasCredentials = preferences.authPersisted && repository.hasStoredToken()
        do {
            let ok = try await repository.checkAuthentication()
            if !ok {
                preferences.authPersisted = false
            }
            return ok
        } catch {
            return hasCredentials
        }
    }

    func fetchAuthInfo() async throws -> AuthInfo {
        try await repository.fetchAuthInfo()
    }

    func logout() async {
        try? await repository.logout()
        preferences.authPersisted = false
        preferences.selectedSessionID = nil
    }
}

