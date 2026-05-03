import Foundation

final class AppPreferences {
    private enum Keys {
        static let authPersisted = "threadgpt_authed"
        static let selectedSessionID = "threadgpt_session_id"
        static let theme = "theme"
        static let serverURL = "threadgpt_server_url"
    }

    private let defaults: UserDefaults

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    var authPersisted: Bool {
        get { defaults.bool(forKey: Keys.authPersisted) }
        set { defaults.set(newValue, forKey: Keys.authPersisted) }
    }

    var selectedSessionID: String? {
        get { defaults.string(forKey: Keys.selectedSessionID) }
        set {
            if let newValue, !newValue.isEmpty {
                defaults.set(newValue, forKey: Keys.selectedSessionID)
            } else {
                defaults.removeObject(forKey: Keys.selectedSessionID)
            }
        }
    }

    var themePreference: ThemePreference {
        get {
            guard let value = defaults.string(forKey: Keys.theme),
                  let theme = ThemePreference(rawValue: value) else {
                return .system
            }
            return theme
        }
        set { defaults.set(newValue.rawValue, forKey: Keys.theme) }
    }

    var serverURL: String {
        get {
            let stored = defaults.string(forKey: Keys.serverURL) ?? ""
            return stored.isEmpty ? ThreadGPTConstants.defaultServerURL : stored
        }
        set { defaults.set(newValue.trimmingCharacters(in: .whitespacesAndNewlines), forKey: Keys.serverURL) }
    }
}

