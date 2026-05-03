import SwiftUI

struct ChatContainerView: View {
    @StateObject private var viewModel: ChatViewModel
    @State private var columnVisibility: NavigationSplitViewVisibility = .automatic
    @State private var sidebarSelection: SidebarDestination?
    @State private var isSyncingSidebarSelection = false

    init(appModel: AppViewModel) {
        _viewModel = StateObject(wrappedValue: ChatViewModel(appModel: appModel))
    }

    var body: some View {
        NavigationSplitView(columnVisibility: $columnVisibility) {
            SessionSidebarView(viewModel: viewModel, selection: $sidebarSelection)
        } detail: {
            ChatScreen(viewModel: viewModel)
        }
        .task {
            await viewModel.loadInitialStateIfNeeded()
        }
        .onChange(of: sidebarSelection) { destination in
            guard !isSyncingSidebarSelection else {
                isSyncingSidebarSelection = false
                return
            }

            handleSidebarSelection(destination)
        }
        .onChange(of: viewModel.selectedSessionId) { sessionId in
            guard let sessionId else {
                return
            }

            let destination = SidebarDestination.session(sessionId)
            guard sidebarSelection != destination else {
                return
            }

            isSyncingSidebarSelection = true
            sidebarSelection = destination
        }
    }

    private func handleSidebarSelection(_ destination: SidebarDestination?) {
        guard let destination else {
            return
        }

        columnVisibility = .detailOnly

        switch destination {
        case .newConversation:
            viewModel.startNewConversation()
        case .session(let sessionId):
            Task {
                await viewModel.loadSession(sessionId)
            }
        }
    }
}
