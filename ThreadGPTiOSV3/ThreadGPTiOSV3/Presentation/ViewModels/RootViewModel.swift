import Foundation

@MainActor
final class RootViewModel: ObservableObject {
    @Published var isLoggedIn = false
    @Published var isCheckingAuth = true

    private let authRepo: AuthRepository

    init(authRepo: AuthRepository = AppContainer.shared.authRepo) {
        self.authRepo = authRepo
    }

    func checkAuth() async {
        guard APIClient.shared.hasToken else {
            isCheckingAuth = false
            return
        }
        do {
            try await authRepo.checkAuth()
            isLoggedIn = true
        } catch {
            APIClient.shared.setToken(nil)
            isLoggedIn = false
        }
        isCheckingAuth = false
    }

    func didLogin() {
        isLoggedIn = true
    }

    func didLogout() {
        isLoggedIn = false
    }
}
