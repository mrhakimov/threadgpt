import SwiftUI

struct MessageListView: View {
    let messages: [Message]
    let streamingContent: String
    let isStreaming: Bool
    let isSending: Bool
    var onFollowUp: ((Message) -> Void)?
    var onEditSystemPrompt: ((String) async -> Bool)?
    var onLoadMore: (() async -> Void)?
    var hasMore: Bool = false
    var preservedContentOffset: Binding<CGPoint?>? = nil

    @State private var bottomDistance: CGFloat = 0
    @State private var scrollButtonGestureActive = false
    @State private var scrollView: UIScrollView?
    @State private var pendingScrollRestore: CGPoint?

    private let bottomID = "message-list-bottom"
    private let autoScrollThreshold: CGFloat = 80
    private let scrollButtonThreshold: CGFloat = 300
    private var hasScrollableContent: Bool {
        !messages.isEmpty || isSending || !streamingContent.isEmpty
    }
    private var showsScrollToBottom: Bool { bottomDistance > scrollButtonThreshold }

    var body: some View {
        GeometryReader { outer in
            ScrollViewReader { proxy in
                ZStack(alignment: .bottom) {
                    ScrollView {
                        VStack(spacing: 0) {
                            LazyVStack(spacing: 0) {
                                if hasMore {
                                    Button("Load more") {
                                        Task { await onLoadMore?() }
                                    }
                                    .font(.caption)
                                    .foregroundColor(.tgptMutedForeground)
                                    .padding(.vertical, 8)
                                }

                                ForEach(messages) { message in
                                    MessageBubbleView(
                                        message: message,
                                        onFollowUp: followUpAction(for: message),
                                        onEditSystemPrompt: message.isSystemPrompt
                                            ? onEditSystemPrompt
                                            : nil
                                    )
                                }

                                // Streaming message
                                if isStreaming && !streamingContent.isEmpty {
                                    MessageBubbleView(
                                        message: Message(
                                            id: "streaming",
                                            sessionId: "",
                                            role: .assistant,
                                            content: streamingContent,
                                            replyCount: 0,
                                            createdAt: ""
                                        ),
                                        isStreaming: true
                                    )
                                }

                                // Sending indicator
                                if isSending && streamingContent.isEmpty {
                                    HStack {
                                        LoadingDotsView()
                                            .padding(.horizontal, 14)
                                            .padding(.vertical, 12)
                                            .background(Color.tgptCard)
                                            .cornerRadius(16)
                                        Spacer()
                                    }
                                    .padding(.horizontal, 16)
                                    .padding(.vertical, 2)
                                }
                            }

                            Color.clear
                                .frame(height: 1)
                                .id(bottomID)
                                .background(
                                    GeometryReader { marker in
                                        Color.clear.preference(
                                            key: MessageListBottomPreferenceKey.self,
                                            value: marker.frame(in: .named("message-list-scroll")).maxY
                                        )
                                    }
                                )
                        }
                        .padding(.top, 8)
                        .padding(.bottom, hasScrollableContent && showsScrollToBottom ? 64 : 16)
                        .background(
                            ScrollViewResolver { resolvedScrollView in
                                if scrollView !== resolvedScrollView {
                                    scrollView = resolvedScrollView
                                }

                                if let pendingScrollRestore {
                                    self.pendingScrollRestore = nil
                                    restoreScrollOffset(pendingScrollRestore)
                                }
                            }
                        )
                    }
                    .coordinateSpace(name: "message-list-scroll")
                    .scrollDismissesKeyboard(.interactively)
                    .onPreferenceChange(MessageListBottomPreferenceKey.self) { bottomMaxY in
                        bottomDistance = max(0, bottomMaxY - outer.size.height)
                    }

                    if hasScrollableContent && showsScrollToBottom {
                        Button {
                            jumpToBottom(proxy)
                        } label: {
                            Image(systemName: "chevron.down")
                                .font(.system(size: 16, weight: .semibold))
                                .foregroundColor(.tgptForeground)
                                .frame(width: 36, height: 36)
                                .background(Color.tgptBackground)
                                .clipShape(Capsule())
                                .overlay(
                                    Capsule()
                                        .stroke(Color.tgptBorder, lineWidth: 1)
                                )
                                .shadow(color: .black.opacity(0.16), radius: 8, x: 0, y: 4)
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel("Scroll to bottom")
                        .accessibilityHint("Jumps to the latest message")
                        .padding(.bottom, 12)
                        .transition(.scale.combined(with: .opacity))
                        .highPriorityGesture(
                            DragGesture(minimumDistance: 0)
                                .onChanged { _ in
                                    guard !scrollButtonGestureActive else { return }
                                    scrollButtonGestureActive = true
                                    jumpToBottom(proxy)
                                }
                                .onEnded { _ in
                                    scrollButtonGestureActive = false
                                    jumpToBottom(proxy)
                                }
                        )
                    }
                }
                .animation(.easeOut(duration: 0.18), value: showsScrollToBottom)
                .onAppear {
                    if let preservedOffset = preservedContentOffset?.wrappedValue {
                        restoreScrollOffset(preservedOffset)
                    } else {
                        scrollToBottom(proxy, animated: false)
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                            scrollToBottom(proxy, animated: false)
                        }
                    }
                }
                .onDisappear {
                    saveScrollOffset()
                }
                .onChange(of: messages.last?.id) {
                    if bottomDistance < autoScrollThreshold {
                        scrollToBottom(proxy, animated: true)
                    }
                }
                .onChange(of: streamingContent) {
                    if bottomDistance < autoScrollThreshold {
                        scrollToBottom(proxy, animated: false)
                    }
                }
            }
        }
    }

    private func jumpToBottom(_ proxy: ScrollViewProxy) {
        dismissKeyboard()
        cancelScrollMomentum()
        scrollToBottom(proxy, animated: true)

        DispatchQueue.main.async {
            cancelScrollMomentum()
            scrollToBottom(proxy, animated: true)
        }

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.08) {
            scrollToBottom(proxy, animated: true)
        }
    }

    private func scrollToBottom(_ proxy: ScrollViewProxy, animated: Bool) {
        if animated {
            withAnimation(.easeOut(duration: 0.28)) {
                proxy.scrollTo(bottomID, anchor: .bottom)
            }
        } else {
            proxy.scrollTo(bottomID, anchor: .bottom)
        }
    }

    private func saveScrollOffset() {
        guard preservedContentOffset != nil, let scrollView else { return }
        preservedContentOffset?.wrappedValue = scrollView.contentOffset
    }

    private func restoreScrollOffset(_ offset: CGPoint) {
        guard let scrollView else {
            pendingScrollRestore = offset
            return
        }

        setScrollOffset(offset, in: scrollView)

        DispatchQueue.main.async {
            setScrollOffset(offset, in: scrollView)
        }

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.06) {
            setScrollOffset(offset, in: scrollView)
        }
    }

    private func setScrollOffset(_ offset: CGPoint, in scrollView: UIScrollView) {
        let minY = -scrollView.adjustedContentInset.top
        let maxY = max(
            minY,
            scrollView.contentSize.height - scrollView.bounds.height + scrollView.adjustedContentInset.bottom
        )
        let clampedY = min(max(offset.y, minY), maxY)

        scrollView.setContentOffset(CGPoint(x: offset.x, y: clampedY), animated: false)
    }

    private func followUpAction(for message: Message) -> (() -> Void)? {
        guard shouldShowFollowUp(for: message), let onFollowUp else {
            return nil
        }

        return { onFollowUp(message) }
    }

    private func shouldShowFollowUp(for message: Message) -> Bool {
        message.role == .assistant && !message.isSystemPromptConfirmation
    }

    private func dismissKeyboard() {
        UIApplication.shared.sendAction(
            #selector(UIResponder.resignFirstResponder),
            to: nil,
            from: nil,
            for: nil
        )
    }

    private func cancelScrollMomentum() {
        guard let scrollView, scrollView.isDecelerating || scrollView.isDragging else {
            return
        }

        scrollView.setContentOffset(scrollView.contentOffset, animated: false)
    }
}

private struct MessageListBottomPreferenceKey: PreferenceKey {
    static var defaultValue: CGFloat = 0

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = nextValue()
    }
}

private struct ScrollViewResolver: UIViewRepresentable {
    let onResolve: (UIScrollView) -> Void

    func makeUIView(context: Context) -> UIView {
        let view = UIView(frame: .zero)
        view.isUserInteractionEnabled = false
        resolve(from: view)
        return view
    }

    func updateUIView(_ view: UIView, context: Context) {
        resolve(from: view)
    }

    private func resolve(from view: UIView) {
        DispatchQueue.main.async {
            guard let scrollView = view.enclosingScrollView() else { return }
            onResolve(scrollView)
        }
    }
}

private extension UIView {
    func enclosingScrollView() -> UIScrollView? {
        var currentView = superview

        while let view = currentView {
            if let scrollView = view as? UIScrollView {
                return scrollView
            }

            currentView = view.superview
        }

        return nil
    }
}
