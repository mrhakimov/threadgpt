import SwiftUI

struct ThreadView: View {
    @Environment(\.dismiss) private var dismiss

    @StateObject private var viewModel: ThreadViewModel
    @State private var copiedPrompt = false

    let onReplySent: () -> Void

    init(
        parentMessage: Message,
        systemPrompt: String?,
        onReplySent: @escaping () -> Void
    ) {
        self._viewModel = StateObject(wrappedValue: ThreadViewModel(
            parentMessage: parentMessage,
            systemPrompt: systemPrompt
        ))
        self.onReplySent = onReplySent
    }

    var body: some View {
        ZStack {
            Color.tgptBackground
                .ignoresSafeArea()

            VStack(spacing: 0) {
                Divider()
                    .background(Color.tgptBorder)

                // System prompt
                systemPromptHeader

                Divider()
                    .background(Color.tgptBorder)

                // Messages
                if !viewModel.hasLoadedMessages || (viewModel.isLoading && viewModel.messages.isEmpty) {
                    Spacer()
                    ProgressView()
                        .tint(.tgptMutedForeground)
                    Spacer()
                } else if viewModel.messages.isEmpty && !viewModel.isSending {
                    threadEmptyState
                } else {
                    MessageListView(
                        messages: viewModel.messages,
                        streamingContent: viewModel.streamingContent,
                        isStreaming: viewModel.isStreaming,
                        isSending: viewModel.isSending,
                        onLoadMore: { await viewModel.loadMore() },
                        hasMore: viewModel.hasMore
                    )
                }

                // Error
                if let error = viewModel.error {
                    Text(error)
                        .font(.caption)
                        .foregroundColor(.tgptDestructive)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 4)
                }

                // Composer
                ComposerView(
                    placeholder: "Ask a follow-up...",
                    disabled: viewModel.isSending
                ) { content in
                    Task { await viewModel.sendReply(content) }
                }
            }
        }
        .navigationTitle("Follow-up")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar(.visible, for: .navigationBar)
        .toolbarBackground(Color.tgptBackground, for: .navigationBar)
        .toolbarBackground(.visible, for: .navigationBar)
        .navigationBarBackButtonHidden(true)
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button {
                    dismiss()
                } label: {
                    Image(systemName: "chevron.left")
                        .font(.system(size: 17, weight: .semibold))
                        .foregroundColor(.tgptForeground)
                        .frame(width: 32, height: 32)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Back")
            }
        }
        .background(InteractivePopGestureEnabler())
        .task {
            viewModel.onReplySent = onReplySent
            await viewModel.loadSystemPrompt()
            await viewModel.loadMessages()
        }
    }

    private var systemPromptHeader: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Image(systemName: "brain")
                    .font(.caption2)
                    .foregroundColor(.tgptMutedForeground)

                Text("System prompt")
                    .font(.caption2)
                    .fontWeight(.semibold)
                    .foregroundColor(.tgptMutedForeground)
                    .textCase(.uppercase)
            }

            if viewModel.isLoadingSystemPrompt && viewModel.systemPrompt.isEmpty {
                HStack(spacing: 6) {
                    ProgressView()
                        .scaleEffect(0.65)
                        .tint(.tgptMutedForeground)
                    Text("Loading system prompt...")
                        .font(.caption)
                        .foregroundColor(.tgptMutedForeground)
                }
            } else {
                Text(viewModel.systemPrompt.isEmpty ? "No system prompt set" : viewModel.systemPrompt)
                    .font(.caption)
                    .foregroundColor(.tgptMutedForeground)
                    .lineLimit(4)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .contentShape(Rectangle())
        .contextMenu {
            if !viewModel.systemPrompt.isEmpty {
                Button {
                    copySystemPrompt()
                } label: {
                    Label(copiedPrompt ? "Copied" : "Copy", systemImage: copiedPrompt ? "checkmark" : "doc.on.doc")
                }
            }
        }
    }

    private var threadEmptyState: some View {
        VStack(spacing: 8) {
            Spacer()
            Image(systemName: "bubble.left")
                .font(.system(size: 24))
                .foregroundColor(.tgptMutedForeground.opacity(0.5))
            Text("Ask a follow-up question below")
                .font(.subheadline)
                .foregroundColor(.tgptMutedForeground)
            Spacer()
        }
    }

    private func copySystemPrompt() {
        UIPasteboard.general.string = viewModel.systemPrompt
        copiedPrompt = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
            copiedPrompt = false
        }
    }
}

private struct InteractivePopGestureEnabler: UIViewControllerRepresentable {
    func makeUIViewController(context: Context) -> UIViewController {
        let controller = UIViewController()
        controller.view.backgroundColor = .clear
        return controller
    }

    func updateUIViewController(_ uiViewController: UIViewController, context: Context) {
        DispatchQueue.main.async {
            guard let navigationController = uiViewController.navigationController else { return }
            navigationController.interactivePopGestureRecognizer?.isEnabled = true
            navigationController.interactivePopGestureRecognizer?.delegate = nil
        }
    }
}
