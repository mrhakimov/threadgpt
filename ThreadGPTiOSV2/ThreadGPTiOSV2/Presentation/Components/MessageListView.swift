import SwiftUI

struct MessageListView: View {
    @Environment(\.colorScheme) private var colorScheme

    let messages: [ChatMessage]
    let streamingContent: String
    let isSending: Bool
    let hasMore: Bool
    let isLoadingMore: Bool
    var showSystemPrompt = true
    var contentAlignment: VerticalAlignment = .bottom
    var onLoadMore: (() -> Void)?
    var onReply: ((ChatMessage) -> Void)?
    var onEditSystemPrompt: ((String) async throws -> Void)?

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(spacing: 14) {
                    if hasMore {
                        Button(action: { onLoadMore?() }) {
                            if isLoadingMore {
                                ProgressView()
                                    .controlSize(.small)
                            } else {
                                Label("Load older messages", systemImage: "arrow.up")
                                    .font(.caption)
                            }
                        }
                        .buttonStyle(.plain)
                        .foregroundStyle(palette.mutedForeground)
                        .padding(.top, 8)
                    }

                    ForEach(messages) { message in
                        MessageBubbleView(
                            message: message,
                            onReply: shouldAllowReply(message) ? onReply : nil,
                            onEditSystemPrompt: shouldAllowSystemPromptEdit(message) ? onEditSystemPrompt : nil
                        )
                        .id(message.id)
                    }

                    if isSending && streamingContent.isEmpty {
                        HStack {
                            LoadingDotsView()
                            Spacer()
                        }
                        .id("__dots__")
                    }

                    if !streamingContent.isEmpty {
                        MessageBubbleView(
                            message: ChatMessage(
                                id: "__streaming__",
                                sessionID: "",
                                role: .assistant,
                                content: streamingContent,
                                replyCount: nil,
                                createdAt: ISO8601DateFormatter().string(from: Date())
                            ),
                            streaming: true
                        )
                        .id("__streaming__")
                    }
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 16)
                .frame(maxWidth: 780)
                .frame(maxWidth: .infinity)
            }
            .background(palette.background)
            .onChange(of: messages.count) { _, _ in scrollToBottom(proxy) }
            .onChange(of: streamingContent) { _, _ in scrollToBottom(proxy) }
        }
    }

    private func shouldAllowReply(_ message: ChatMessage) -> Bool {
        !message.isSystemPrompt && !message.isSystemPromptConfirmation && message.canStartThread
    }

    private func shouldAllowSystemPromptEdit(_ message: ChatMessage) -> Bool {
        showSystemPrompt && message.isSystemPrompt
    }

    private func scrollToBottom(_ proxy: ScrollViewProxy) {
        DispatchQueue.main.async {
            withAnimation(.easeOut(duration: 0.22)) {
                if !streamingContent.isEmpty {
                    proxy.scrollTo("__streaming__", anchor: .bottom)
                } else if let last = messages.last {
                    proxy.scrollTo(last.id, anchor: .bottom)
                }
            }
        }
    }
}
