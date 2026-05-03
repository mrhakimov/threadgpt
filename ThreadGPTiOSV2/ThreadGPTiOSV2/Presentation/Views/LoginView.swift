import SwiftUI

struct LoginView: View {
    @Environment(\.colorScheme) private var colorScheme
    @EnvironmentObject private var rootViewModel: RootViewModel
    @StateObject var viewModel: LoginViewModel

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        VStack {
            Spacer()

            VStack(alignment: .leading, spacing: 30) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("ThreadGPT")
                        .font(.system(size: 28, weight: .semibold))
                        .foregroundStyle(palette.foreground)
                    Text("Each message gets its own isolated context - no bloat, ever.")
                        .font(.system(size: 15))
                        .foregroundStyle(palette.mutedForeground)
                }

                VStack(alignment: .leading, spacing: 14) {
                    VStack(alignment: .leading, spacing: 7) {
                        Text("OpenAI API Key")
                            .font(.caption.weight(.medium))
                            .textCase(.uppercase)
                            .tracking(1.5)
                            .foregroundStyle(palette.mutedForeground)
                        SecureField("sk-...", text: $viewModel.apiKey)
                            .font(.system(.body, design: .monospaced))
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .padding(.vertical, 10)
                            .overlay(alignment: .bottom) { palette.border.frame(height: 1) }
                    }

                    VStack(alignment: .leading, spacing: 7) {
                        Text("Server")
                            .font(.caption.weight(.medium))
                            .textCase(.uppercase)
                            .tracking(1.5)
                            .foregroundStyle(palette.mutedForeground)
                        TextField("http://localhost:8000", text: $rootViewModel.serverURL)
                            .keyboardType(.URL)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                            .font(.system(.footnote, design: .monospaced))
                            .padding(.vertical, 8)
                            .overlay(alignment: .bottom) { palette.border.frame(height: 1) }
                            .onSubmit { rootViewModel.setServerURL(rootViewModel.serverURL) }
                            .onChange(of: rootViewModel.serverURL) { _, value in rootViewModel.setServerURL(value) }
                    }

                    if let error = viewModel.errorMessage {
                        Text(error)
                            .font(.caption)
                            .foregroundStyle(palette.destructive)
                    }

                    Button(action: viewModel.submit) {
                        HStack {
                            Spacer()
                            if viewModel.isLoading {
                                ProgressView()
                                    .tint(palette.background)
                            } else {
                                Text("Continue")
                                    .fontWeight(.semibold)
                            }
                            Spacer()
                        }
                        .padding(.vertical, 12)
                    }
                    .buttonStyle(.plain)
                    .background(palette.foreground)
                    .foregroundStyle(palette.background)
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                    .disabled(!viewModel.canSubmit)
                    .opacity(viewModel.canSubmit ? 1 : 0.5)
                }

                VStack(alignment: .leading, spacing: 14) {
                    Hairline()
                    Text("Your key is encrypted server-side for your session. Depending on how this server is configured, encrypted session data may also be persisted locally to survive restarts. Your raw API key is never stored in the database.")
                        .font(.caption)
                        .foregroundStyle(palette.mutedForeground)
                        .lineSpacing(2)
                }

                HStack(spacing: 8) {
                    Text("Built by")
                    Link("@omtiness", destination: URL(string: "https://x.com/omtiness")!)
                    Text("-")
                    Link("GitHub", destination: URL(string: "https://github.com/mrhakimov/threadgpt")!)
                }
                .font(.caption)
                .foregroundStyle(palette.mutedForeground)
            }
            .frame(maxWidth: 390)
            .padding(.horizontal, 24)

            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(palette.background.ignoresSafeArea())
    }
}
