import SwiftUI

struct MessageBubbleView: View {
    let message: Message
    var isStreaming = false
    var onReply: ((Message) -> Void)?

    private var canReply: Bool {
        message.isAssistant && !isStreaming && !message.id.hasPrefix("system-prompt") && onReply != nil
    }

    var body: some View {
        HStack(alignment: .top) {
            if message.isUser {
                Spacer(minLength: 36)
            }

            VStack(alignment: message.isUser ? .trailing : .leading, spacing: 6) {
                Text(message.isUser ? "You" : "ThreadGPT")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)

                Text(message.content)
                    .textSelection(.enabled)
                    .font(.body)
                    .frame(maxWidth: .infinity, alignment: message.isUser ? .trailing : .leading)
                    .padding(12)
                    .background(bubbleBackground)
                    .foregroundStyle(message.isUser ? .white : .primary)
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))

                if canReply {
                    Button {
                        onReply?(message)
                    } label: {
                        Label(threadLabel, systemImage: "arrowshape.turn.up.left")
                    }
                    .font(.caption)
                    .buttonStyle(.borderless)
                }
            }
            .frame(maxWidth: 620, alignment: message.isUser ? .trailing : .leading)

            if message.isAssistant {
                Spacer(minLength: 36)
            }
        }
    }

    private var bubbleBackground: some ShapeStyle {
        if message.isUser {
            return AnyShapeStyle(Color.accentColor)
        }
        return AnyShapeStyle(Color(uiColor: .secondarySystemBackground))
    }

    private var threadLabel: String {
        guard let count = message.replyCount, count > 0 else {
            return "Thread"
        }

        return "Thread (\(count))"
    }
}
