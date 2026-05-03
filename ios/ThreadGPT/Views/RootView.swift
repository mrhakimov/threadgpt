import SwiftUI

struct RootView: View {
    @EnvironmentObject private var appModel: AppViewModel

    var body: some View {
        switch appModel.phase {
        case .loading:
            VStack(spacing: 16) {
                ProgressView()
                Text("ThreadGPT")
                    .font(.headline)
            }
        case .signedOut:
            LoginView()
        case .signedIn:
            ChatContainerView(appModel: appModel)
        }
    }
}
