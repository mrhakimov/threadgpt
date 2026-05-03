import Foundation

@MainActor
final class LoginViewModel: ObservableObject {
    @Published var apiKey = ""
    @Published var isLoading = false
    @Published var error: String?

    private let authRepo: AuthRepository

    init(authRepo: AuthRepository = AppContainer.shared.authRepo) {
        self.authRepo = authRepo
    }

    var canSubmit: Bool {
        apiKey.hasPrefix("sk-") && apiKey.count >= 20 && !isLoading
    }

    func login() async -> Bool {
        guard canSubmit else {
            error = "Enter a valid OpenAI API key (starts with sk-)"
            return false
        }

        isLoading = true
        error = nil

        do {
            _ = try await authRepo.login(apiKey: apiKey)
            isLoading = false
            return true
        } catch let appError as AppError {
            error = appError.errorDescription
            isLoading = false
            return false
        } catch {
            self.error = error.localizedDescription
            isLoading = false
            return false
        }
    }
}
