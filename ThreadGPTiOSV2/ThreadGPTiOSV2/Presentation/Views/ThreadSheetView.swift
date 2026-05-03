import SwiftUI

struct ThreadSheetView: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    @StateObject var viewModel: ThreadViewModel

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        VStack(spacing: 0) {
            HStack {
                Text("Thread")
                    .font(.system(size: 17, weight: .semibold))
                Spacer()
                Button(action: { dismiss() }) {
                    Image(systemName: "xmark")
                        .font(.system(size: 15, weight: .semibold))
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 13)
            Hairline()

            VStack(alignment: .leading, spacing: 8) {
                Text("Following up on")
                    .font(.caption.weight(.medium))
                    .textCase(.uppercase)
                    .tracking(1.2)
                    .foregroundStyle(palette.mutedForeground.opacity(0.7))
                HStack(alignment: .top, spacing: 10) {
                    Capsule()
                        .fill(palette.mutedForeground.opacity(0.25))
                        .frame(width: 3)
                    Text(viewModel.parentMessage.content)
                        .font(.system(size: 14))
                        .foregroundStyle(palette.mutedForeground)
                        .lineLimit(4)
                        .lineSpacing(2)
                }
            }
            .padding(16)
            Hairline()

            ZStack {
                if viewModel.isLoading {
                    ProgressView()
                } else if viewModel.messages.isEmpty && viewModel.streamingContent.isEmpty && !viewModel.isSending {
                    Text("Ask a follow-up question below.")
                        .font(.system(size: 14))
                        .foregroundStyle(palette.mutedForeground)
                        .padding(.top, 26)
                        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
                } else {
                    MessageListView(
                        messages: viewModel.messages,
                        streamingContent: viewModel.streamingContent,
                        isSending: viewModel.isSending,
                        hasMore: viewModel.hasMore,
                        isLoadingMore: viewModel.isLoadingMore,
                        showSystemPrompt: false,
                        onLoadMore: viewModel.loadMore
                    )
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

            VStack(spacing: 8) {
                if let error = viewModel.errorMessage {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(palette.destructive)
                        .frame(maxWidth: 520, alignment: .leading)
                }
                ComposerView(
                    placeholder: "Ask a follow-up...",
                    isDisabled: viewModel.isSending,
                    onSend: viewModel.sendMessage
                )
                .frame(maxWidth: 520)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .overlay(alignment: .top) { Hairline() }
        }
        .background(palette.background.ignoresSafeArea())
        .onAppear { viewModel.load() }
        .onDisappear { viewModel.cancel() }
    }
}

