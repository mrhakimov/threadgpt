import SwiftUI

struct LoginView: View {
    @EnvironmentObject private var appModel: AppViewModel

    @State private var apiKey = ""
    @State private var serverURL = AppConfig.defaultBackendURL
    @State private var isSigningIn = false

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Backend URL", text: $serverURL)
                        .keyboardType(.URL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()

                    SecureField("OpenAI API key", text: $apiKey)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()

                    Button {
                        Task {
                            await signIn()
                        }
                    } label: {
                        if isSigningIn {
                            ProgressView()
                        } else {
                            Label("Sign in", systemImage: "key.fill")
                        }
                    }
                    .disabled(isSigningIn || apiKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }

                if let authError = appModel.authError {
                    Section {
                        Text(authError)
                            .foregroundStyle(.red)
                    }
                }
            }
            .navigationTitle("ThreadGPT")
            .onAppear {
                serverURL = appModel.serverURLString
            }
        }
    }

    private func signIn() async {
        isSigningIn = true
        defer { isSigningIn = false }

        do {
            try appModel.updateServerURL(serverURL)
            await appModel.signIn(apiKey: apiKey.trimmingCharacters(in: .whitespacesAndNewlines))
        } catch {
            appModel.authError = appModel.errorMessage(from: error)
        }
    }
}
