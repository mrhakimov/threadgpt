import Foundation
import SwiftUI

@MainActor
final class RootViewModel: ObservableObject {
    enum AuthState: Equatable {
        case checking
        case loggedOut
        case loggedIn
    }

    @Published var authState: AuthState = .checking
    @Published var selectedSessionID: String?
    @Published var themePreference: ThemePreference
    @Published var serverURL: String

    private let authService: AuthService
    private let preferences: AppPreferences

    init(authService: AuthService, preferences: AppPreferences) {
        self.authService = authService
        self.preferences = preferences
        self.selectedSessionID = preferences.selectedSessionID
        self.themePreference = preferences.themePreference
        self.serverURL = preferences.serverURL
    }

    var preferredColorScheme: ColorScheme? {
        themePreference.preferredColorScheme
    }

    func bootstrap() {
        Task {
            let ok = await authService.checkAuthorization()
            authState = ok ? .loggedIn : .loggedOut
            selectedSessionID = ok ? preferences.selectedSessionID : nil
        }
    }

    func login(apiKey: String) async throws {
        try await authService.authenticate(apiKey: apiKey)
        selectSession(nil)
        authState = .loggedIn
    }

    func logout() {
        Task {
            await authService.logout()
            selectSession(nil)
            authState = .loggedOut
        }
    }

    func handleUnauthorized() {
        Task {
            await authService.logout()
            selectSession(nil)
            authState = .loggedOut
        }
    }

    func selectSession(_ sessionID: String?) {
        selectedSessionID = sessionID
        preferences.selectedSessionID = sessionID
    }

    func setTheme(_ preference: ThemePreference) {
        themePreference = preference
        preferences.themePreference = preference
    }

    func setServerURL(_ value: String) {
        serverURL = value
        preferences.serverURL = value
    }
}

