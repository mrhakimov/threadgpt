import Foundation

@MainActor
final class AuthStore: ObservableObject {
    @Published private(set) var token: String?

    init() {
        token = KeychainService.readToken()
    }

    func setToken(_ token: String) {
        self.token = token
        KeychainService.saveToken(token)
    }

    func clearToken() {
        token = nil
        KeychainService.deleteToken()
    }
}
