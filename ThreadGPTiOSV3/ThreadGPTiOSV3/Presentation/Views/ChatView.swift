import SwiftUI

struct ChatView: View {
    @StateObject private var chatVM = ChatViewModel()
    @StateObject private var sidebarVM = ConversationListViewModel()
    @State private var activeSessionId: String?
    @State private var showSidebar = false
    @State private var showSettings = false
    @State private var threadParent: Message?
    @State private var showThread = false
    @State private var threadScrollOffset: CGPoint?

    let onUnauthorized: () -> Void

    var body: some View {
        NavigationStack {
            chatContent
                .toolbar(.hidden, for: .navigationBar)
                .navigationDestination(isPresented: $showThread) {
                    if let parent = threadParent {
                        ThreadView(
                            parentMessage: parent,
                            systemPrompt: chatVM.session?.systemPrompt,
                            onReplySent: {
                                chatVM.incrementReplyCount(for: parent.id)
                            }
                        )
                    }
                }
        }
        .sheet(isPresented: $showSidebar) {
            ConversationListView(
                viewModel: sidebarVM,
                activeSessionId: activeSessionId,
                onSelect: { sessionId in
                    showSidebar = false
                    selectSession(sessionId)
                },
                onNew: {
                    showSidebar = false
                    startNewConversation()
                },
                onRename: { id, name in
                    if id == activeSessionId {
                        chatVM.session?.name = name
                    }
                },
                onDelete: { id in
                    if id == activeSessionId {
                        startNewConversation()
                    }
                }
            )
        }
        .sheet(isPresented: $showSettings) {
            SettingsView(onLogout: {
                showSettings = false
                onUnauthorized()
            })
        }
        .task {
            chatVM.onUnauthorized = onUnauthorized
            sidebarVM.onUnauthorized = onUnauthorized
            await sidebarVM.loadSessions()
        }
    }

    private var chatContent: some View {
        ZStack {
            Color.tgptBackground
                .ignoresSafeArea()

            VStack(spacing: 0) {
                // Header
                chatHeader

                // Content
                if chatVM.isLoading && chatVM.messages.isEmpty {
                    Spacer()
                    ProgressView()
                        .tint(.tgptMutedForeground)
                    Spacer()
                } else if chatVM.displayMessages.isEmpty && !chatVM.isSending {
                    emptyState
                } else {
                    MessageListView(
                        messages: chatVM.displayMessages,
                        streamingContent: chatVM.streamingContent,
                        isStreaming: chatVM.isStreaming,
                        isSending: chatVM.isSending,
                        onFollowUp: { message in
                            threadParent = message
                            showThread = true
                        },
                        onEditSystemPrompt: { prompt in
                            let updated = await chatVM.updateSystemPrompt(prompt)
                            if updated {
                                await sidebarVM.refresh()
                            }
                            return updated
                        },
                        onLoadMore: { await chatVM.loadMore() },
                        hasMore: chatVM.hasMore,
                        preservedContentOffset: $threadScrollOffset
                    )
                }

                // Error
                if let error = chatVM.error {
                    Text(error)
                        .font(.caption)
                        .foregroundColor(.tgptDestructive)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 4)
                }

                // Composer
                ComposerView(
                    placeholder: chatVM.isFirstMessage
                        ? "Set your assistant's context..."
                        : "Send a message...",
                    disabled: chatVM.isSending
                ) { content in
                    Task {
                        let newId = await chatVM.sendMessage(content)
                        if let newId, activeSessionId == nil {
                            activeSessionId = newId
                            await sidebarVM.refresh()
                        }
                    }
                }
            }
        }
    }

    // MARK: - Header

    private var chatHeader: some View {
        HStack(spacing: 12) {
            Button(action: { showSidebar = true }) {
                Image(systemName: "line.3.horizontal")
                    .font(.system(size: 18, weight: .medium))
                    .foregroundColor(.tgptForeground)
            }

            VStack(alignment: .leading, spacing: 1) {
                Text("ThreadGPT")
                    .font(.headline)
                    .foregroundColor(.tgptForeground)

                if let name = chatVM.conversationName {
                    Text(name)
                        .font(.caption)
                        .foregroundColor(.tgptMutedForeground)
                        .lineLimit(1)
                }
            }

            Spacer()

            Button(action: startNewConversation) {
                Image(systemName: "plus")
                    .font(.system(size: 16, weight: .medium))
                    .foregroundColor(.tgptForeground)
            }

            Button(action: { showSettings = true }) {
                Image(systemName: "gearshape")
                    .font(.system(size: 16, weight: .medium))
                    .foregroundColor(.tgptForeground)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .background(Color.tgptBackground)
        .overlay(
            Divider().background(Color.tgptBorder),
            alignment: .bottom
        )
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: 12) {
            Spacer()

            Image(systemName: "bubble.left.and.bubble.right")
                .font(.system(size: 32))
                .foregroundColor(.tgptMutedForeground.opacity(0.5))

            Text(chatVM.isFirstMessage
                 ? "Start by setting your assistant's context"
                 : "Send a message to start chatting")
                .font(.subheadline)
                .foregroundColor(.tgptMutedForeground)
                .multilineTextAlignment(.center)

            if chatVM.isFirstMessage {
                Text("Your first message becomes the assistant's instructions")
                    .font(.caption)
                    .foregroundColor(.tgptMutedForeground.opacity(0.7))
            }

            Spacer()
        }
        .padding(.horizontal, 32)
    }

    // MARK: - Actions

    private func selectSession(_ sessionId: String) {
        activeSessionId = sessionId
        threadScrollOffset = nil
        Task { await chatVM.loadSession(sessionId) }
    }

    private func startNewConversation() {
        activeSessionId = nil
        threadScrollOffset = nil
        Task { await chatVM.loadSession(nil) }
    }
}
