import SwiftUI

struct ThreadSheetView: View {
    @Environment(\.dismiss) private var dismiss

    @StateObject private var viewModel: ThreadViewModel
    @State private var composerText = ""

    private let onReplySent: () -> Void

    init(parentMessage: Message, appModel: AppViewModel, onReplySent: @escaping () -> Void) {
        _viewModel = StateObject(wrappedValue: ThreadViewModel(parentMessage: parentMessage, appModel: appModel))
        self.onReplySent = onReplySent
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                parentPreview

                Divider()

                content

                if let errorMessage = viewModel.errorMessage {
                    Text(errorMessage)
                        .font(.footnote)
                        .foregroundStyle(.red)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal)
                        .padding(.vertical, 8)
                        .background(.red.opacity(0.08))
                }

                ComposerView(
                    text: $composerText,
                    isSending: viewModel.isSending,
                    placeholder: "Ask a follow-up..."
                ) {
                    let text = composerText
                    composerText = ""

                    Task {
                        await viewModel.sendMessage(text, onReplySent: onReplySent)
                    }
                }
                .padding()
            }
            .navigationTitle("Thread")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
            .task {
                await viewModel.load()
            }
        }
    }

    private var parentPreview: some View {
        VStack(alignment: .leading, spacing: 6) {
            Label("Following up on", systemImage: "quote.bubble")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            Text(viewModel.parentMessage.content)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .lineLimit(5)
                .textSelection(.enabled)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
    }

    @ViewBuilder
    private var content: some View {
        if viewModel.isLoading {
            VStack {
                Spacer()
                ProgressView()
                Spacer()
            }
        } else if viewModel.messages.isEmpty && viewModel.streamingContent.isEmpty {
            VStack(spacing: 12) {
                Image(systemName: "arrowshape.turn.up.left")
                    .font(.system(size: 32))
                    .foregroundStyle(.secondary)
                Text("Ask a follow-up below.")
                    .font(.headline)
                    .foregroundStyle(.secondary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else {
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 12) {
                        if viewModel.hasMoreMessages {
                            Button {
                                Task {
                                    await viewModel.loadMoreMessages()
                                }
                            } label: {
                                Label("Load earlier", systemImage: "chevron.up")
                            }
                            .buttonStyle(.bordered)
                        }

                        ForEach(viewModel.messages) { message in
                            MessageBubbleView(message: message)
                                .id(message.id)
                        }

                        if !viewModel.streamingContent.isEmpty {
                            MessageBubbleView(
                                message: Message(
                                    id: "thread-streaming-assistant",
                                    sessionId: viewModel.parentMessage.sessionId,
                                    role: "assistant",
                                    content: viewModel.streamingContent,
                                    replyCount: nil,
                                    createdAt: ""
                                ),
                                isStreaming: true
                            )
                            .id("thread-streaming-assistant")
                        }
                    }
                    .padding()
                }
                .onChange(of: viewModel.messages.count) { _ in
                    scrollToBottom(proxy)
                }
                .onChange(of: viewModel.streamingContent) { _ in
                    scrollToBottom(proxy)
                }
            }
        }
    }

    private func scrollToBottom(_ proxy: ScrollViewProxy) {
        let target = viewModel.streamingContent.isEmpty
            ? viewModel.messages.last?.id
            : "thread-streaming-assistant"

        guard let target else {
            return
        }

        withAnimation(.easeOut(duration: 0.2)) {
            proxy.scrollTo(target, anchor: .bottom)
        }
    }
}
