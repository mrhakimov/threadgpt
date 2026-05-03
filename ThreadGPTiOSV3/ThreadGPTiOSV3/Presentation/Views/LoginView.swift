import SwiftUI

struct LoginView: View {
    @StateObject private var viewModel = LoginViewModel()
    let onLogin: () -> Void

    var body: some View {
        ZStack {
            Color.tgptBackground
                .ignoresSafeArea()

            VStack(spacing: 24) {
                Spacer()

                // Logo
                VStack(spacing: 8) {
                    Image(systemName: "bubble.left.and.bubble.right")
                        .font(.system(size: 40))
                        .foregroundColor(.tgptForeground)

                    Text("ThreadGPT")
                        .font(.title)
                        .fontWeight(.bold)
                        .foregroundColor(.tgptForeground)

                    Text("Chat without context bloat")
                        .font(.subheadline)
                        .foregroundColor(.tgptMutedForeground)
                }

                // Input
                VStack(spacing: 12) {
                    SecureField("sk-...", text: $viewModel.apiKey)
                        .textFieldStyle(.plain)
                        .padding(14)
                        .background(Color.tgptSecondary)
                        .cornerRadius(10)
                        .font(.body.monospaced())
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)

                    if let error = viewModel.error {
                        Text(error)
                            .font(.caption)
                            .foregroundColor(.tgptDestructive)
                            .multilineTextAlignment(.center)
                    }

                    Button(action: {
                        Task {
                            if await viewModel.login() {
                                onLogin()
                            }
                        }
                    }) {
                        Group {
                            if viewModel.isLoading {
                                ProgressView()
                                    .tint(.tgptBackground)
                            } else {
                                Text("Sign in")
                                    .fontWeight(.semibold)
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .foregroundColor(.tgptBackground)
                        .background(viewModel.canSubmit ? Color.tgptForeground : Color.tgptMutedForeground)
                        .cornerRadius(10)
                    }
                    .disabled(!viewModel.canSubmit)
                }
                .padding(.horizontal, 32)

                Spacer()

                // Footer
                VStack(spacing: 4) {
                    Text("Your API key is encrypted and never stored in plain text.")
                        .font(.caption2)
                        .foregroundColor(.tgptMutedForeground)
                    Text("Session expires after 24 hours.")
                        .font(.caption2)
                        .foregroundColor(.tgptMutedForeground)
                }
                .padding(.bottom, 24)
            }
        }
    }
}
