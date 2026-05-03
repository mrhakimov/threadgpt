import SwiftUI

@main
struct ThreadGPTiOSV2App: App {
    @StateObject private var container = AppContainer()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(container)
                .environmentObject(container.rootViewModel)
        }
    }
}

