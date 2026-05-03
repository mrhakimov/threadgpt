import Foundation

@MainActor
final class SettingsViewModel: ObservableObject {
    @Published var expiresAt: Date?
    @Published var isLoading = false
    @Published var models: [String] = []
    @Published var selectedModel: String = ""
    @Published var isLoadingModels = false

    private let authRepo: AuthRepository
    private let api = APIClient.shared

    init(authRepo: AuthRepository = AppContainer.shared.authRepo) {
        self.authRepo = authRepo
        self.selectedModel = ModelPreference.shared.selectedModel ?? ""
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

    func loadModels() async {
        isLoadingModels = true
        do {
            let response: ModelsResponse = try await api.request(method: "GET", path: "/api/models")
            models = response.models.sorted()
            if selectedModel.isEmpty, !models.isEmpty {
                selectModel(models.first(where: { $0 == "gpt-4o" }) ?? models[0])
            }
        } catch {}
        isLoadingModels = false
    }

    func selectModel(_ model: String) {
        selectedModel = model
        ModelPreference.shared.selectedModel = model
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

private struct ModelsResponse: Codable {
    let models: [String]
}
