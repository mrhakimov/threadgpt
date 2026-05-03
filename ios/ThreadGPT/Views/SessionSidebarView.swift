import SwiftUI

enum SidebarDestination: Hashable {
    case newConversation
    case session(String)
}

struct SessionSidebarView: View {
    @ObservedObject var viewModel: ChatViewModel
    @Binding var selection: SidebarDestination?

    @State private var renameCandidate: SessionInfo?
    @State private var renameText = ""
    @State private var deleteCandidate: SessionInfo?

    var body: some View {
        List(selection: $selection) {
            NavigationLink(value: SidebarDestination.newConversation) {
                Label("New conversation", systemImage: "square.and.pencil")
            }

            if viewModel.isLoadingSessions {
                HStack {
                    Spacer()
                    ProgressView()
                    Spacer()
                }
            }

            ForEach(viewModel.sessions) { session in
                if let sessionId = session.sessionId {
                    NavigationLink(value: SidebarDestination.session(sessionId)) {
                        SessionRow(
                            session: session,
                            isSelected: sessionId == viewModel.selectedSessionId
                        )
                    }
                    .swipeActions(edge: .trailing) {
                        Button(role: .destructive) {
                            deleteCandidate = session
                        } label: {
                            Label("Delete", systemImage: "trash")
                        }

                        Button {
                            renameCandidate = session
                            renameText = session.displayTitle
                        } label: {
                            Label("Rename", systemImage: "pencil")
                        }
                        .tint(.blue)
                    }
                } else {
                    SessionRow(
                        session: session,
                        isSelected: false
                    )
                }
            }
        }
        .navigationTitle("Conversations")
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button {
                    Task {
                        await viewModel.refreshSessions()
                    }
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
            }
        }
        .refreshable {
            await viewModel.refreshSessions()
        }
        .alert("Rename conversation", isPresented: Binding(
            get: { renameCandidate != nil },
            set: { if !$0 { renameCandidate = nil } }
        )) {
            TextField("Name", text: $renameText)
            Button("Save") {
                guard let sessionId = renameCandidate?.sessionId else {
                    return
                }

                Task {
                    await viewModel.renameSession(sessionId: sessionId, name: renameText)
                    renameCandidate = nil
                }
            }
            Button("Cancel", role: .cancel) {
                renameCandidate = nil
            }
        }
        .confirmationDialog(
            "Delete conversation?",
            isPresented: Binding(
                get: { deleteCandidate != nil },
                set: { if !$0 { deleteCandidate = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button("Delete", role: .destructive) {
                guard let sessionId = deleteCandidate?.sessionId else {
                    return
                }

                Task {
                    await viewModel.deleteSession(sessionId: sessionId)
                    deleteCandidate = nil
                }
            }
            Button("Cancel", role: .cancel) {
                deleteCandidate = nil
            }
        }
    }
}

private struct SessionRow: View {
    let session: SessionInfo
    let isSelected: Bool

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: isSelected ? "bubble.left.and.bubble.right.fill" : "bubble.left.and.bubble.right")
                .foregroundStyle(isSelected ? Color.accentColor : Color.secondary)

            VStack(alignment: .leading, spacing: 4) {
                Text(session.displayTitle)
                    .lineLimit(2)
                    .font(.body.weight(isSelected ? .semibold : .regular))

                if let createdAt = session.createdAt, !createdAt.isEmpty {
                    Text(createdAt)
                        .lineLimit(1)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(.vertical, 4)
        .contentShape(Rectangle())
    }
}
