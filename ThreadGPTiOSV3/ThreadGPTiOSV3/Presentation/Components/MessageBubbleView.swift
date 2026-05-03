import SwiftUI

struct MessageBubbleView: View {
    let message: Message
    var isStreaming = false
    var onFollowUp: (() -> Void)?
    var onEditSystemPrompt: ((String) async -> Bool)?

    @State private var copied = false
    @State private var showSystemPromptEditor = false
    @State private var systemPromptDraft = ""
    @State private var isSavingSystemPrompt = false
    @State private var systemPromptSaveError: String?

    private var isUser: Bool { message.role == .user }
    private var bubbleBackground: Color {
        if message.isSystemPrompt {
            return Color.tgptCard.opacity(0.72)
        }
        return isUser ? Color.tgptSecondary : Color.tgptCard
    }

    private var bubbleShape: RoundedRectangle {
        RoundedRectangle(cornerRadius: 16, style: .continuous)
    }

    var body: some View {
        HStack(alignment: .top, spacing: 0) {
            if isUser { Spacer(minLength: 48) }

            VStack(alignment: isUser ? .trailing : .leading, spacing: 6) {
                // Bubble
                Text(message.content)
                    .font(.body)
                    .foregroundColor(.tgptForeground)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 10)
                    .background(bubbleBackground)
                    .clipShape(bubbleShape)
                    .overlay {
                        if message.isSystemPrompt {
                            bubbleShape
                                .stroke(Color.tgptBorder, lineWidth: 1)
                        }
                    }
                    .contextMenu {
                        Button {
                            UIPasteboard.general.string = message.content
                            copied = true
                            DispatchQueue.main.asyncAfter(deadline: .now() + 2) { copied = false }
                        } label: {
                            Label(copied ? "Copied" : "Copy", systemImage: copied ? "checkmark" : "doc.on.doc")
                        }

                        if message.isSystemPrompt, onEditSystemPrompt != nil {
                            Button {
                                beginEditingSystemPrompt()
                            } label: {
                                Label("Edit", systemImage: "pencil")
                            }
                        }
                    }

                // Streaming cursor
                if isStreaming {
                    HStack(spacing: 4) {
                        Circle()
                            .fill(Color.tgptMutedForeground)
                            .frame(width: 6, height: 6)
                            .opacity(0.6)
                        Text("Typing...")
                            .font(.caption2)
                            .foregroundColor(.tgptMutedForeground)
                    }
                    .padding(.leading, 4)
                }

                // Follow-up button for assistant messages
                if !isUser && !isStreaming && onFollowUp != nil {
                    Button(action: { onFollowUp?() }) {
                        HStack(spacing: 4) {
                            Image(systemName: "bubble.left")
                                .font(.caption)
                            Text("Follow up")
                                .font(.caption)
                            if message.replyCount > 0 {
                                Text("\(message.replyCount)")
                                    .font(.caption2)
                                    .fontWeight(.semibold)
                                    .foregroundColor(.white)
                                    .padding(.horizontal, 5)
                                    .padding(.vertical, 1)
                                    .background(Color.tgptMutedForeground)
                                    .cornerRadius(8)
                            }
                        }
                        .foregroundColor(.tgptMutedForeground)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                    }
                }
            }

            if !isUser { Spacer(minLength: 48) }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 2)
        .sheet(isPresented: $showSystemPromptEditor) {
            systemPromptEditor
        }
    }

    private var systemPromptEditor: some View {
        NavigationStack {
            VStack(spacing: 10) {
                TextEditor(text: $systemPromptDraft)
                    .font(.body)
                    .foregroundColor(.tgptForeground)
                    .padding(8)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                    .background(Color.tgptCard)
                    .cornerRadius(8)
                    .scrollContentBackground(.hidden)
                    .disabled(isSavingSystemPrompt)

                if let systemPromptSaveError {
                    Text(systemPromptSaveError)
                        .font(.caption)
                        .foregroundColor(.tgptDestructive)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .padding(16)
            .background(Color.tgptBackground.ignoresSafeArea())
            .navigationTitle("System prompt")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        showSystemPromptEditor = false
                    }
                    .disabled(isSavingSystemPrompt)
                }

                ToolbarItem(placement: .confirmationAction) {
                    Button(isSavingSystemPrompt ? "Saving" : "Save") {
                        Task { await saveSystemPrompt() }
                    }
                    .disabled(isSavingSystemPrompt || systemPromptDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private func beginEditingSystemPrompt() {
        systemPromptDraft = message.content
        systemPromptSaveError = nil
        showSystemPromptEditor = true
    }

    @MainActor
    private func saveSystemPrompt() async {
        let trimmed = systemPromptDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, let onEditSystemPrompt else { return }

        systemPromptSaveError = nil
        isSavingSystemPrompt = true
        let updated = await onEditSystemPrompt(trimmed)
        isSavingSystemPrompt = false

        if updated {
            showSystemPromptEditor = false
        } else {
            systemPromptSaveError = "Could not update system prompt"
        }
    }
}
