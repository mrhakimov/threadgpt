import SwiftUI

struct ConversationListView: View {
    @ObservedObject var viewModel: ConversationListViewModel
    let activeSessionId: String?
    let onSelect: (String) -> Void
    let onNew: () -> Void
    let onRename: (String, String) -> Void
    let onDelete: (String) -> Void

    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ZStack {
                Color.tgptBackground
                    .ignoresSafeArea()

                if viewModel.isLoading && viewModel.sessions.isEmpty {
                    ProgressView()
                        .tint(.tgptMutedForeground)
                } else if viewModel.sessions.isEmpty {
                    VStack(spacing: 8) {
                        Image(systemName: "tray")
                            .font(.system(size: 28))
                            .foregroundColor(.tgptMutedForeground.opacity(0.5))
                        Text("No conversations yet")
                            .font(.subheadline)
                            .foregroundColor(.tgptMutedForeground)
                    }
                } else {
                    sessionList
                }
            }
            .navigationTitle("Conversations")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Close") { dismiss() }
                        .foregroundColor(.tgptForeground)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button(action: onNew) {
                        Image(systemName: "plus")
                            .foregroundColor(.tgptForeground)
                    }
                }
            }
        }
    }

    private var sessionList: some View {
        List {
            ForEach(viewModel.sessions) { session in
                sessionRow(session)
                    .listRowBackground(
                        session.sessionId == activeSessionId
                            ? Color.tgptSecondary
                            : Color.clear
                    )
                    .listRowInsets(EdgeInsets(top: 4, leading: 16, bottom: 4, trailing: 16))
            }

            if viewModel.hasMore {
                Button("Load more") {
                    Task { await viewModel.loadMore() }
                }
                .font(.caption)
                .foregroundColor(.tgptMutedForeground)
                .frame(maxWidth: .infinity)
                .listRowBackground(Color.clear)
            }
        }
        .listStyle(.plain)
        .scrollContentBackground(.hidden)
        .refreshable {
            await viewModel.refresh()
        }
    }

    @ViewBuilder
    private func sessionRow(_ session: Session) -> some View {
        if viewModel.editingId == session.sessionId {
            // Editing mode
            HStack(spacing: 8) {
                TextField("Name", text: $viewModel.editingName)
                    .textFieldStyle(.plain)
                    .font(.body)
                    .foregroundColor(.tgptForeground)
                    .onSubmit {
                        Task {
                            if let id = await viewModel.commitRename() {
                                onRename(id, viewModel.editingName)
                            }
                        }
                    }

                Button {
                    Task {
                        if let id = await viewModel.commitRename() {
                            onRename(id, viewModel.editingName)
                        }
                    }
                } label: {
                    Image(systemName: "checkmark")
                        .font(.caption)
                        .foregroundColor(.tgptForeground)
                }
            }
        } else if viewModel.confirmDeleteId == session.sessionId {
            // Delete confirmation
            HStack {
                Text("Delete?")
                    .font(.body)
                    .foregroundColor(.tgptForeground)
                Spacer()
                Button("Yes") {
                    guard let id = session.sessionId else { return }
                    Task {
                        await viewModel.deleteSession(id)
                        onDelete(id)
                    }
                }
                .foregroundColor(.tgptDestructive)
                .fontWeight(.semibold)

                Button("No") {
                    viewModel.confirmDeleteId = nil
                }
                .foregroundColor(.tgptMutedForeground)
            }
        } else {
            // Normal row
            Button {
                if let id = session.sessionId { onSelect(id) }
            } label: {
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(session.name ?? session.systemPrompt ?? "New conversation")
                            .font(.body)
                            .foregroundColor(.tgptForeground)
                            .lineLimit(1)

                        if let date = session.createdAt {
                            Text(formatDate(date))
                                .font(.caption2)
                                .foregroundColor(.tgptMutedForeground)
                        }
                    }
                    Spacer()
                }
            }
            .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                Button(role: .destructive) {
                    viewModel.confirmDeleteId = session.sessionId
                } label: {
                    Image(systemName: "trash")
                }

                Button {
                    viewModel.startRename(session)
                } label: {
                    Image(systemName: "pencil")
                }
                .tint(Color.tgptMutedForeground)
            }
        }
    }

    private func formatDate(_ isoString: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        guard let date = formatter.date(from: isoString) else {
            let basic = ISO8601DateFormatter()
            guard let d = basic.date(from: isoString) else { return "" }
            return RelativeDateTimeFormatter().localizedString(for: d, relativeTo: Date())
        }
        return RelativeDateTimeFormatter().localizedString(for: date, relativeTo: Date())
    }
}
