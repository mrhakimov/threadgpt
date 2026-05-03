import SwiftUI

@main
struct ThreadGPTApp: App {
    @StateObject private var authStore: AuthStore
    @StateObject private var appModel: AppViewModel

    init() {
        let authStore = AuthStore()
        _authStore = StateObject(wrappedValue: authStore)
        _appModel = StateObject(wrappedValue: AppViewModel(authStore: authStore))
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(authStore)
                .environmentObject(appModel)
                .task {
                    await appModel.bootstrap()
                }
        }
    }
}
