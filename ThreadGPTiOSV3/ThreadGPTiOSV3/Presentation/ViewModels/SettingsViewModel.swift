import Foundation

@MainActor
final class SettingsViewModel: ObservableObject {
    @Published var expiresAt: Date?
    @Published var isLoading = false
    @Published var confirmLogout = false

    private let authRepo: AuthRepository

    init(authRepo: AuthRepository = AppContainer.shared.authRepo) {
        self.authRepo = authRepo
    }

    var timeRemaining: String {
        guard let expiresAt else { return "Unknown" }
        let remaining = expiresAt.timeIntervalSinceNow
        if remaining <= 0 { return "Expired" }
        let hours = Int(remaining) / 3600
        let minutes = (Int(remaining) % 3600) / 60
        if hours > 0 {
            return "\(hours)h \(minutes)m"
        }
        return "\(minutes)m"
    }

    func loadInfo() async {
        do {
            let info = try await authRepo.fetchAuthInfo()
            let formatter = ISO8601DateFormatter()
            formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            expiresAt = formatter.date(from: info.expiresAt)
            if expiresAt == nil {
                let basic = ISO8601DateFormatter()
                expiresAt = basic.date(from: info.expiresAt)
            }
        } catch {}
    }

    func logout() async -> Bool {
        isLoading = true
        do {
            try await authRepo.logout()
            isLoading = false
            return true
        } catch {
            isLoading = false
            return false
        }
    }
}
