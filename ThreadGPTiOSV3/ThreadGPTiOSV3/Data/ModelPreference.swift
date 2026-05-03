import Foundation

final class ModelPreference {
    static let shared = ModelPreference()

    private let key = "selectedModel"

    var selectedModel: String? {
        get { UserDefaults.standard.string(forKey: key) }
        set { UserDefaults.standard.set(newValue, forKey: key) }
    }

    private init() {}
}
