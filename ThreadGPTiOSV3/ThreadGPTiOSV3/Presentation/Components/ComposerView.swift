import SwiftUI

struct ComposerView: View {
    let placeholder: String
    let disabled: Bool
    let onSend: (String) -> Void

    @State private var text = ""
    @FocusState private var isFocused: Bool

    var body: some View {
        VStack(spacing: 0) {
            Divider()
                .background(Color.tgptBorder)

            HStack(alignment: .bottom, spacing: 8) {
                TextField(placeholder, text: $text, axis: .vertical)
                    .textFieldStyle(.plain)
                    .lineLimit(1...6)
                    .focused($isFocused)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 10)
                    .background(Color.tgptSecondary)
                    .cornerRadius(20)
                    .onSubmit {
                        send()
                    }

                Button(action: send) {
                    Image(systemName: "arrow.up")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundColor(canSend ? Color.tgptBackground : Color.tgptMutedForeground)
                        .frame(width: 40, height: 40)
                        .background(canSend ? Color.tgptForeground : Color.tgptSecondary)
                        .clipShape(Circle())
                }
                .disabled(!canSend)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
        }
        .background(Color.tgptBackground)
    }

    private var canSend: Bool {
        !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !disabled
    }

    private func send() {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        onSend(trimmed)
        text = ""
    }
}
