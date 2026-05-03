import SwiftUI

struct ConversationListView: View {
    @Environment(\.colorScheme) private var colorScheme
    @ObservedObject var viewModel: ConversationListViewModel
    let activeSessionID: String?
    let isCurrentEmpty: Bool
    let onSelect: (String?) -> Void
    let onNewConversation: () -> Void

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                Image(systemName: "sidebar.left")
                    .font(.system(size: 18, weight: .medium))
                Text("Conversations")
                    .font(.system(size: 15, weight: .semibold))
                Spacer()
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 13)
            Hairline()

            Button(action: onNewConversation) {
                Label("New conversation", systemImage: "plus")
                    .font(.system(size: 14, weight: .medium))
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 9)
                    .overlay(
                        RoundedRectangle(cornerRadius: 8, style: .continuous)
                            .stroke(palette.border, lineWidth: 1)
                    )
            }
            .buttonStyle(.plain)
            .padding(10)

            if let error = viewModel.errorMessage {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(palette.destructive)
                    .padding(.horizontal, 12)
                    .padding(.bottom, 6)
            }

            ScrollView {
                LazyVStack(spacing: 4) {
                    if viewModel.isLoading {
                        ProgressView()
                            .padding(.top, 18)
                    } else if viewModel.sessions.isEmpty {
                        Text("No conversations yet")
                            .font(.caption)
                            .foregroundStyle(palette.mutedForeground)
                            .padding(.top, 24)
                    } else {
                        ForEach(viewModel.sessions) { session in
                            ConversationRow(
                                session: session,
                                active: session.sessionID == activeSessionID,
                                onSelect: { onSelect(session.sessionID) },
                                onRename: { viewModel.beginRename(session) },
                                onDelete: { viewModel.deleteTarget = session }
                            )
                            .onAppear { viewModel.loadMoreIfNeeded(current: session) }
                        }
                    }

                    if viewModel.isLoadingMore {
                        ProgressView()
                            .controlSize(.small)
                            .padding(.vertical, 10)
                    }
                }
                .padding(.horizontal, 8)
                .padding(.bottom, 12)
            }
        }
        .background(palette.background)
        .onAppear { viewModel.load() }
        .alert("Rename conversation", isPresented: Binding(
            get: { viewModel.renameTarget != nil },
            set: { if !$0 { viewModel.renameTarget = nil } }
        )) {
            TextField("Name", text: $viewModel.renameText)
            Button("Cancel", role: .cancel) { viewModel.renameTarget = nil }
            Button("Save") { viewModel.commitRename() }
        }
        .confirmationDialog("Delete this conversation?", isPresented: Binding(
            get: { viewModel.deleteTarget != nil },
            set: { if !$0 { viewModel.deleteTarget = nil } }
        )) {
            Button("Delete", role: .destructive) {
                if let session = viewModel.deleteTarget {
                    viewModel.delete(session)
                }
            }
            Button("Cancel", role: .cancel) {}
        }
    }
}

private struct ConversationRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let session: ChatSession
    let active: Bool
    let onSelect: () -> Void
    let onRename: () -> Void
    let onDelete: () -> Void

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        Button(action: onSelect) {
            Text(getSessionLabel(session))
                .font(.system(size: 14, weight: active ? .semibold : .regular))
                .lineLimit(1)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, 10)
                .padding(.vertical, 10)
                .background(active ? palette.muted : Color.clear)
                .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        }
        .buttonStyle(.plain)
        .contextMenu {
            Button(action: onRename) {
                Label("Rename", systemImage: "pencil")
            }
            Button(role: .destructive, action: onDelete) {
                Label("Delete", systemImage: "trash")
            }
        }
        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
            Button(role: .destructive, action: onDelete) {
                Label("Delete", systemImage: "trash")
            }
            Button(action: onRename) {
                Label("Rename", systemImage: "pencil")
            }
            .tint(.gray)
        }
    }
}
