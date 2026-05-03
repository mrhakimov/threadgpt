import SwiftUI

struct RootView: View {
    @StateObject private var viewModel = RootViewModel()
    @AppStorage("appearance") private var appearance = Appearance.system

    var body: some View {
        Group {
            if viewModel.isCheckingAuth {
                ZStack {
                    Color.tgptBackground.ignoresSafeArea()
                    ProgressView()
                        .tint(.tgptMutedForeground)
                }
            } else if viewModel.isLoggedIn {
                ChatView(onUnauthorized: { viewModel.didLogout() })
            } else {
                LoginView(onLogin: { viewModel.didLogin() })
            }
        }
        .task {
            await viewModel.checkAuth()
        }
        .onAppear {
            applyAppearance(appearance)
        }
    }

    private func applyAppearance(_ appearance: Appearance) {
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene else { return }
        for window in windowScene.windows {
            window.overrideUserInterfaceStyle = appearance.uiStyle
        }
    }
}
