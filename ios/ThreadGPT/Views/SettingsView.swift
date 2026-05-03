import SwiftUI

struct SettingsView: View {
    @Environment(\.dismiss) private var dismiss

    @ObservedObject var appModel: AppViewModel
    @ObservedObject var chatViewModel: ChatViewModel

    @State private var serverURL: String
    @State private var systemPrompt: String
    @State private var statusMessage: String?

    init(appModel: AppViewModel, chatViewModel: ChatViewModel) {
        self.appModel = appModel
        self.chatViewModel = chatViewModel
        _serverURL = State(initialValue: appModel.serverURLString)
        _systemPrompt = State(initialValue: chatViewModel.session?.systemPrompt ?? "")
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("Backend") {
                    TextField("Backend URL", text: $serverURL)
                        .keyboardType(.URL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()

                    Button {
                        saveServerURL()
                    } label: {
                        Label("Save URL", systemImage: "checkmark.circle")
                    }
                }

                Section("Conversation") {
                    TextEditor(text: $systemPrompt)
                        .frame(minHeight: 120)
                        .disabled(chatViewModel.session?.sessionId == nil)

                    Button {
                        Task {
                            await saveSystemPrompt()
                        }
                    } label: {
                        Label("Save instructions", systemImage: "text.badge.checkmark")
                    }
                    .disabled(chatViewModel.session?.sessionId == nil)
                }

                Section("Account") {
                    if let expiresAt = appModel.authInfo?.expiresAt {
                        LabeledContent("Session expires", value: expiresAt)
                    }

                    Button(role: .destructive) {
                        Task {
                            await appModel.signOut()
                            dismiss()
                        }
                    } label: {
                        Label("Sign out", systemImage: "rectangle.portrait.and.arrow.right")
                    }
                }

                if let statusMessage {
                    Section {
                        Text(statusMessage)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
        }
    }

    private func saveServerURL() {
        do {
            try appModel.updateServerURL(serverURL)
            statusMessage = "URL saved."
        } catch {
            statusMessage = appModel.errorMessage(from: error)
        }
    }

    private func saveSystemPrompt() async {
        await chatViewModel.updateSystemPrompt(systemPrompt)
        statusMessage = chatViewModel.errorMessage == nil ? "Instructions saved." : chatViewModel.errorMessage
    }
}
