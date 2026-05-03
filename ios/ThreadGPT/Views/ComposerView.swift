import SwiftUI

struct ComposerView: View {
    @Binding var text: String
    let isSending: Bool
    let placeholder: String
    let onSend: () -> Void

    var body: some View {
        HStack(alignment: .bottom, spacing: 10) {
            TextField(placeholder, text: $text, axis: .vertical)
                .lineLimit(1...6)
                .textFieldStyle(.roundedBorder)
                .disabled(isSending)
                .submitLabel(.send)
                .onSubmit {
                    if canSend {
                        onSend()
                    }
                }

            Button {
                onSend()
            } label: {
                if isSending {
                    ProgressView()
                } else {
                    Image(systemName: "arrow.up.circle.fill")
                        .font(.system(size: 28))
                }
            }
            .disabled(!canSend)
        }
    }

    private var canSend: Bool {
        !isSending && !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }
}
