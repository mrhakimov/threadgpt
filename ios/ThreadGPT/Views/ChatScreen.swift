import SwiftUI

struct ChatScreen: View {
    @ObservedObject var viewModel: ChatViewModel

    @State private var composerText = ""
    @State private var activeSheet: ActiveSheet?

    var body: some View {
        VStack(spacing: 0) {
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
                placeholder: viewModel.isFirstMessage ? "Set the instructions..." : "Message ThreadGPT"
            ) {
                let text = composerText
                composerText = ""

                Task {
                    await viewModel.sendMessage(text)
                }
            }
            .padding()
            .background(.background)
        }
        .navigationTitle("ThreadGPT")
        .toolbar {
            ToolbarItem(placement: .principal) {
                VStack(spacing: 2) {
                    Text("ThreadGPT")
                        .font(.headline)
                    Text(viewModel.title)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
            }

            ToolbarItem(placement: .navigationBarTrailing) {
                Button {
                    activeSheet = .settings
                } label: {
                    Image(systemName: "gearshape")
                }
            }
        }
        .sheet(item: $activeSheet) { sheet in
            switch sheet {
            case .thread(let message):
                ThreadSheetView(
                    parentMessage: message,
                    appModel: viewModel.appModel
                ) {
                    viewModel.incrementReplyCount(for: message.id)
                }
            case .settings:
                SettingsView(appModel: viewModel.appModel, chatViewModel: viewModel)
            }
        }
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
                Image(systemName: viewModel.isFirstMessage ? "text.badge.checkmark" : "bubble.left.and.bubble.right")
                    .font(.system(size: 36))
                    .foregroundStyle(.secondary)

                Text(viewModel.isFirstMessage ? "Set this conversation's instructions." : "Start a new turn.")
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
                            MessageBubbleView(message: message) { selected in
                                activeSheet = .thread(selected)
                            }
                            .id(message.id)
                        }

                        if !viewModel.streamingContent.isEmpty {
                            MessageBubbleView(
                                message: Message(
                                    id: "streaming-assistant",
                                    sessionId: viewModel.session?.sessionId ?? "",
                                    role: "assistant",
                                    content: viewModel.streamingContent,
                                    replyCount: nil,
                                    createdAt: ""
                                ),
                                isStreaming: true,
                                onReply: nil
                            )
                            .id("streaming-assistant")
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
            : "streaming-assistant"

        guard let target else {
            return
        }

        withAnimation(.easeOut(duration: 0.2)) {
            proxy.scrollTo(target, anchor: .bottom)
        }
    }
}

private enum ActiveSheet: Identifiable {
    case thread(Message)
    case settings

    var id: String {
        switch self {
        case .thread(let message):
            return "thread-\(message.id)"
        case .settings:
            return "settings"
        }
    }
}
