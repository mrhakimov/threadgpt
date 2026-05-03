import Foundation

@MainActor
final class SettingsViewModel: ObservableObject {
    @Published private(set) var isLoading = false
    @Published private(set) var expiresAt: Date?
    @Published var serverURL: String

    private let authService: AuthService
    private let rootViewModel: RootViewModel
    private let preferences: AppPreferences

    init(authService: AuthService, rootViewModel: RootViewModel, preferences: AppPreferences) {
        self.authService = authService
        self.rootViewModel = rootViewModel
        self.preferences = preferences
        self.serverURL = preferences.serverURL
    }

    var timeRemaining: String {
        guard let expiresAt else { return "" }
        let seconds = Int(expiresAt.timeIntervalSinceNow)
        guard seconds > 0 else { return "Expired" }
        let minutes = seconds / 60
        let hours = minutes / 60
        if hours > 0 {
            return "\(hours)h \(minutes % 60)m remaining"
        }
        return "\(minutes)m remaining"
    }

    func load() {
        Task {
            isLoading = true
            if let info = try? await authService.fetchAuthInfo() {
                expiresAt = ISO8601DateFormatter().date(from: info.expiresAt)
            }
            isLoading = false
        }
    }

    func saveServerURL() {
        rootViewModel.setServerURL(serverURL)
        preferences.serverURL = serverURL
    }

    func logout() {
        rootViewModel.logout()
    }
}

