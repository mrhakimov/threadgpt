import SwiftUI
import UIKit

struct MessageBubbleView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var copied = false
    @State private var editing = false
    @State private var editText: String
    @State private var saveError: String?

    let message: ChatMessage
    var streaming = false
    var onReply: ((ChatMessage) -> Void)?
    var onEditSystemPrompt: ((String) async throws -> Void)?

    init(
        message: ChatMessage,
        streaming: Bool = false,
        onReply: ((ChatMessage) -> Void)? = nil,
        onEditSystemPrompt: ((String) async throws -> Void)? = nil
    ) {
        self.message = message
        self.streaming = streaming
        self.onReply = onReply
        self.onEditSystemPrompt = onEditSystemPrompt
        _editText = State(initialValue: message.content)
    }

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        HStack {
            if !message.isAssistant { Spacer(minLength: 40) }

            VStack(alignment: message.isAssistant ? .leading : .trailing, spacing: 4) {
                bubble(palette: palette)
                    .contextMenu {
                        Button(action: copy) {
                            Label("Copy", systemImage: copied ? "checkmark" : "doc.on.doc")
                        }
                        if message.isSystemPrompt, onEditSystemPrompt != nil {
                            Button(action: { editing = true }) {
                                Label("Edit system prompt", systemImage: "pencil")
                            }
                        }
                    }

                if message.canStartThread, let onReply {
                    Button(action: { onReply(message) }) {
                        Label(followUpTitle, systemImage: "message")
                            .font(.system(size: 12, weight: .medium))
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(palette.mutedForeground)
                    .padding(.leading, 4)
                    .padding(.vertical, 2)
                }
            }
            .frame(maxWidth: message.isAssistant ? .infinity : UIScreen.main.bounds.width * 0.8, alignment: message.isAssistant ? .leading : .trailing)

            if message.isAssistant { Spacer(minLength: 40) }
        }
    }

    @ViewBuilder
    private func bubble(palette: ThreadGPTPalette) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            if editing {
                TextEditor(text: $editText)
                    .font(.system(size: 15))
                    .scrollContentBackground(.hidden)
                    .frame(minHeight: 92)
                    .foregroundStyle(palette.foreground)
            } else {
                Text(message.content)
                    .font(.system(size: 15))
                    .lineSpacing(3)
                    .foregroundStyle(textColor(palette: palette))
                    .textSelection(.enabled)
            }

            if let saveError {
                Text(saveError)
                    .font(.caption2)
                    .foregroundStyle(palette.destructive)
            }

            HStack(spacing: 12) {
                if message.isAssistant && !streaming {
                    Button(action: copy) {
                        Image(systemName: copied ? "checkmark" : "doc.on.doc")
                    }
                }

                if message.isSystemPrompt {
                    if editing {
                        Button(action: cancelEdit) { Image(systemName: "xmark") }
                        Button(action: saveEdit) { Image(systemName: "checkmark") }
                    } else {
                        Button(action: { editing = true }) { Image(systemName: "pencil") }
                        Button(action: copy) { Image(systemName: copied ? "checkmark" : "doc.on.doc") }
                    }
                    Image(systemName: "info.circle")
                        .accessibilityLabel("system prompt")
                }

                if streaming {
                    Rectangle()
                        .fill(textColor(palette: palette).opacity(0.7))
                        .frame(width: 2, height: 16)
                        .opacity(0.8)
                }
            }
            .font(.system(size: 13, weight: .medium))
            .foregroundStyle(palette.mutedForeground)
            .frame(maxWidth: .infinity, alignment: .trailing)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(backgroundColor(palette: palette))
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(message.isSystemPrompt ? palette.border : .clear, lineWidth: 1)
        )
    }

    private var followUpTitle: String {
        let count = message.replyCount ?? 0
        if count == 1 { return "1 follow-up" }
        if count > 1 { return "\(count) follow-ups" }
        return "Follow up"
    }

    private func backgroundColor(palette: ThreadGPTPalette) -> Color {
        if message.isAssistant { return palette.muted }
        if message.isSystemPrompt { return palette.muted.opacity(0.6) }
        return palette.secondary
    }

    private func textColor(palette: ThreadGPTPalette) -> Color {
        message.isAssistant ? palette.foreground : palette.foreground
    }

    private func copy() {
        UIPasteboard.general.string = message.content
        copied = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.4) {
            copied = false
        }
    }

    private func cancelEdit() {
        editText = message.content
        saveError = nil
        editing = false
    }

    private func saveEdit() {
        guard let onEditSystemPrompt else {
            editing = false
            return
        }
        Task {
            do {
                try await onEditSystemPrompt(editText.trimmingCharacters(in: .whitespacesAndNewlines))
                editing = false
                saveError = nil
            } catch {
                saveError = error.localizedDescription
            }
        }
    }
}

