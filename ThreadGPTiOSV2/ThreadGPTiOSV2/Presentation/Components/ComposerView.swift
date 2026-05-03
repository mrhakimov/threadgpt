import SwiftUI

struct ComposerView: View {
    @Environment(\.colorScheme) private var colorScheme
    @FocusState private var focused: Bool
    @State private var text = ""

    let placeholder: String
    let isDisabled: Bool
    let onSend: (String) -> Void

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        VStack(alignment: .leading, spacing: 8) {
            ZStack(alignment: .topLeading) {
                if text.isEmpty {
                    Text(placeholder)
                        .font(.system(size: 15))
                        .foregroundStyle(palette.mutedForeground)
                        .padding(.top, 9)
                        .padding(.leading, 5)
                }

                TextEditor(text: $text)
                    .focused($focused)
                    .font(.system(size: 15))
                    .scrollContentBackground(.hidden)
                    .foregroundStyle(palette.foreground)
                    .frame(minHeight: 40, maxHeight: 130)
                    .disabled(isDisabled)
                    .submitLabel(.send)
            }

            HStack {
                Spacer()
                Button(action: send) {
                    Image(systemName: "arrow.up")
                        .font(.system(size: 15, weight: .semibold))
                        .frame(width: 30, height: 30)
                        .background(palette.foreground.opacity(canSend ? 0.15 : 0.05))
                        .foregroundStyle(palette.foreground.opacity(canSend ? 1 : 0.25))
                        .clipShape(Circle())
                }
                .buttonStyle(.plain)
                .disabled(!canSend)
                .accessibilityLabel("Send message")
            }
        }
        .padding(.horizontal, 14)
        .padding(.top, 10)
        .padding(.bottom, 10)
        .background(palette.muted.opacity(0.55))
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .stroke(palette.border.opacity(0.8), lineWidth: 1)
        )
        .shadow(color: .black.opacity(colorScheme == .dark ? 0.25 : 0.08), radius: 8, y: 2)
        .onTapGesture { focused = true }
    }

    private var canSend: Bool {
        !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isDisabled
    }

    private func send() {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, !isDisabled else { return }
        onSend(trimmed)
        text = ""
    }
}

