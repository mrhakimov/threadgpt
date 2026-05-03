import Foundation

@MainActor
final class LoginViewModel: ObservableObject {
    @Published var apiKey = ""
    @Published var isLoading = false
    @Published var errorMessage: String?

    private let rootViewModel: RootViewModel

    init(rootViewModel: RootViewModel) {
        self.rootViewModel = rootViewModel
    }

    var canSubmit: Bool {
        !apiKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isLoading
    }

    func submit() {
        let trimmed = apiKey.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }

        Task {
            isLoading = true
            errorMessage = nil
            do {
                try await rootViewModel.login(apiKey: trimmed)
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }
}

