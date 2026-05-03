import SwiftUI

struct RootView: View {
    @EnvironmentObject private var container: AppContainer
    @EnvironmentObject private var rootViewModel: RootViewModel
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        Group {
            switch rootViewModel.authState {
            case .checking:
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(palette.background)
            case .loggedOut:
                LoginView(viewModel: container.makeLoginViewModel())
            case .loggedIn:
                ChatShellView(container: container)
            }
        }
        .preferredColorScheme(rootViewModel.preferredColorScheme)
        .tint(palette.foreground)
        .onAppear { rootViewModel.bootstrap() }
    }
}

