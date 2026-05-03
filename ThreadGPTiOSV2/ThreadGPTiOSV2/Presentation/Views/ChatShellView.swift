import SwiftUI

struct ChatShellView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(\.colorScheme) private var colorScheme
    @EnvironmentObject private var rootViewModel: RootViewModel
    @EnvironmentObject private var container: AppContainer
    @StateObject private var chatViewModel: ChatViewModel
    @StateObject private var conversationListViewModel: ConversationListViewModel
    @State private var showingConversations = false
    @State private var showingSettings = false
    @State private var threadParent: ChatMessage?

    init(container: AppContainer) {
        _chatViewModel = StateObject(wrappedValue: container.makeChatViewModel())
        _conversationListViewModel = StateObject(wrappedValue: container.makeConversationListViewModel())
    }

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        Group {
            if horizontalSizeClass == .compact {
                ChatDetailView(
                    chatViewModel: chatViewModel,
                    onOpenConversations: { showingConversations = true },
                    onOpenSettings: { showingSettings = true },
                    onOpenThread: { threadParent = $0 }
                )
                .sheet(isPresented: $showingConversations) {
                    ConversationListView(
                        viewModel: conversationListViewModel,
                        activeSessionID: rootViewModel.selectedSessionID,
                        isCurrentEmpty: chatViewModel.isEmpty,
                        onSelect: { sessionID in
                            rootViewModel.selectSession(sessionID)
                            showingConversations = false
                        },
                        onNewConversation: {
                            if chatViewModel.isEmpty {
                                showingConversations = false
                            } else {
                                rootViewModel.selectSession(nil)
                                showingConversations = false
                            }
                        }
                    )
                    .presentationDetents([.medium, .large])
                }
            } else {
                HStack(spacing: 0) {
                    ConversationListView(
                        viewModel: conversationListViewModel,
                        activeSessionID: rootViewModel.selectedSessionID,
                        isCurrentEmpty: chatViewModel.isEmpty,
                        onSelect: rootViewModel.selectSession,
                        onNewConversation: {
                            if !chatViewModel.isEmpty {
                                rootViewModel.selectSession(nil)
                            }
                        }
                    )
                    .frame(width: 270)
                    palette.border.frame(width: 1)

                    ChatDetailView(
                        chatViewModel: chatViewModel,
                        onOpenConversations: { showingConversations = true },
                        onOpenSettings: { showingSettings = true },
                        onOpenThread: { threadParent = $0 }
                    )
                }
            }
        }
        .background(palette.background.ignoresSafeArea())
        .onAppear {
            configureCallbacks()
            chatViewModel.selectSession(rootViewModel.selectedSessionID)
        }
        .onChange(of: rootViewModel.selectedSessionID) { _, newValue in
            chatViewModel.selectSession(newValue)
        }
        .sheet(isPresented: $showingSettings) {
            SettingsView(viewModel: container.makeSettingsViewModel())
        }
        .sheet(item: $threadParent) { parent in
            ThreadSheetView(viewModel: container.makeThreadViewModel(parentMessage: parent) {
                chatViewModel.incrementReplyCount(for: parent.id, by: 1)
            })
        }
    }

    private func configureCallbacks() {
        chatViewModel.onSessionResolved = { sessionID in
            rootViewModel.selectSession(sessionID)
        }
        chatViewModel.onUnauthorized = {
            rootViewModel.handleUnauthorized()
        }
        chatViewModel.onConversationMutated = {
            conversationListViewModel.load()
        }
        conversationListViewModel.onRenamed = { sessionID, name in
            chatViewModel.updateLocalSessionName(sessionID: sessionID, name: name)
        }
        conversationListViewModel.onDeleted = { sessionID in
            if rootViewModel.selectedSessionID == sessionID {
                rootViewModel.selectSession(nil)
            }
        }
    }
}

private struct ChatDetailView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(\.colorScheme) private var colorScheme
    @ObservedObject var chatViewModel: ChatViewModel
    let onOpenConversations: () -> Void
    let onOpenSettings: () -> Void
    let onOpenThread: (ChatMessage) -> Void

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        VStack(spacing: 0) {
            header(palette: palette)
            Hairline()

            ZStack {
                if chatViewModel.isLoading {
                    ProgressView()
                } else if chatViewModel.isEmpty {
                    emptyState(palette: palette)
                } else {
                    MessageListView(
                        messages: chatViewModel.messages,
                        streamingContent: chatViewModel.streamingContent,
                        isSending: chatViewModel.isSending,
                        hasMore: chatViewModel.hasMoreMessages,
                        isLoadingMore: chatViewModel.isLoadingMore,
                        onLoadMore: chatViewModel.loadMoreMessages,
                        onReply: onOpenThread,
                        onEditSystemPrompt: { content in
                            try await chatViewModel.updateSystemPrompt(content)
                        }
                    )
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(palette.background)

            VStack(spacing: 8) {
                if let error = chatViewModel.errorMessage {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(palette.destructive)
                        .frame(maxWidth: 780, alignment: .leading)
                }
                ComposerView(
                    placeholder: chatViewModel.composerPlaceholder,
                    isDisabled: chatViewModel.isSending,
                    onSend: chatViewModel.sendMessage
                )
                .frame(maxWidth: 780)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(palette.background)
            .overlay(alignment: .top) { Hairline() }
        }
    }

    private func header(palette: ThreadGPTPalette) -> some View {
        HStack(spacing: 12) {
            if horizontalSizeClass == .compact {
                Button(action: onOpenConversations) {
                    Image(systemName: "sidebar.left")
                }
                .buttonStyle(.plain)
            }

            VStack(alignment: .leading, spacing: 2) {
                Button(action: chatViewModel.loadAllMessages) {
                    Text("ThreadGPT")
                        .font(.system(size: 17, weight: .semibold))
                }
                .buttonStyle(.plain)

                if let subtitle = chatViewModel.subtitle {
                    Button(action: chatViewModel.loadAllMessages) {
                        Text(subtitle)
                            .font(.caption)
                            .foregroundStyle(palette.mutedForeground)
                            .lineLimit(1)
                    }
                    .buttonStyle(.plain)
                }
            }

            Spacer()

            Link(destination: URL(string: "https://x.com/omtiness")!) {
                Text("X")
                    .font(.system(size: 14, weight: .semibold))
            }
            .buttonStyle(.plain)
            .foregroundStyle(palette.mutedForeground)

            Link(destination: URL(string: "https://github.com/mrhakimov/threadgpt")!) {
                Image(systemName: "link")
            }
            .buttonStyle(.plain)
            .foregroundStyle(palette.mutedForeground)

            Button(action: onOpenSettings) {
                Image(systemName: "gearshape")
            }
            .buttonStyle(.plain)
            .foregroundStyle(palette.foreground)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(palette.background)
    }

    private func emptyState(palette: ThreadGPTPalette) -> some View {
        VStack(spacing: 10) {
            Text(chatViewModel.isFirstMessage ? "Set your conversation context" : "Start chatting")
                .font(.system(size: 19, weight: .medium))
                .foregroundStyle(palette.foreground)
            Text(chatViewModel.isFirstMessage
                 ? "Your first message becomes the assistant's instructions for this entire conversation. Make it count!"
                 : "Send a message to continue your conversation.")
                .font(.system(size: 14))
                .foregroundStyle(palette.mutedForeground)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 360)
        }
        .padding(.horizontal, 24)
    }
}
