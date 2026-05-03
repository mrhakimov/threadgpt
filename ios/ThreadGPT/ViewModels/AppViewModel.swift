import Foundation

enum AppPhase: Equatable {
    case loading
    case signedOut
    case signedIn
}

@MainActor
final class AppViewModel: ObservableObject {
    @Published var phase: AppPhase = .loading
    @Published var serverURLString: String
    @Published var authError: String?
    @Published var authInfo: AuthInfo?

    private let authStore: AuthStore

    init(authStore: AuthStore) {
        self.authStore = authStore
        serverURLString = UserDefaults.standard.string(forKey: AppConfig.serverURLDefaultsKey)
            ?? AppConfig.defaultBackendURL
    }

    func bootstrap() async {
        authError = nil

        guard authStore.token != nil else {
            phase = .signedOut
            return
        }

        do {
            let client = try makeClient()
            if await client.checkAuthentication() {
                authInfo = try? await client.fetchAuthInfo()
                phase = .signedIn
            } else {
                authStore.clearToken()
                phase = .signedOut
            }
        } catch {
            authError = errorMessage(from: error)
            phase = .signedOut
        }
    }

    func signIn(apiKey: String) async {
        authError = nil

        do {
            let url = try normalizedServerURL()
            let client = APIClient(baseURL: url, bearerToken: nil)
            let response = try await client.authenticate(apiKey: apiKey)

            authStore.setToken(response.token)
            authInfo = try? await makeClient().fetchAuthInfo()
            phase = .signedIn
        } catch {
            authError = errorMessage(from: error)
            phase = .signedOut
        }
    }

    func signOut() async {
        if let client = try? makeClient() {
            await client.logout()
        }

        authStore.clearToken()
        authInfo = nil
        phase = .signedOut
    }

    func updateServerURL(_ value: String) throws {
        let url = try normalizedServerURL(from: value)
        let normalized = url.absoluteString.trimmingCharacters(in: CharacterSet(charactersIn: "/"))

        serverURLString = normalized
        UserDefaults.standard.set(normalized, forKey: AppConfig.serverURLDefaultsKey)
    }

    func makeClient() throws -> APIClient {
        try APIClient(baseURL: normalizedServerURL(), bearerToken: authStore.token)
    }

    func errorMessage(from error: Error) -> String {
        if let localized = error as? LocalizedError, let message = localized.errorDescription {
            return message
        }
        return error.localizedDescription
    }

    private func normalizedServerURL() throws -> URL {
        try normalizedServerURL(from: serverURLString)
    }

    private func normalizedServerURL(from value: String) throws -> URL {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
            .trimmingCharacters(in: CharacterSet(charactersIn: "/"))

        guard
            let url = URL(string: trimmed),
            let scheme = url.scheme?.lowercased(),
            ["http", "https"].contains(scheme),
            url.host != nil
        else {
            throw APIServiceError.invalidServerURL
        }

        return url
    }
}
