import SwiftUI

struct SettingsView: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    @EnvironmentObject private var rootViewModel: RootViewModel
    @StateObject var viewModel: SettingsViewModel
    @State private var confirmingLogout = false

    var body: some View {
        let palette = ThreadGPTPalette.resolve(colorScheme)
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 30) {
                    sectionTitle("Appearance", palette: palette)
                    HStack {
                        Text("Dark mode")
                            .font(.system(size: 15))
                        Spacer()
                        Picker("Dark mode", selection: Binding(
                            get: { rootViewModel.themePreference },
                            set: { rootViewModel.setTheme($0) }
                        )) {
                            ForEach(ThemePreference.allCases) { theme in
                                Text(theme.title).tag(theme)
                            }
                        }
                        .pickerStyle(.segmented)
                        .frame(width: 230)
                    }

                    sectionTitle("API Key", palette: palette)
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Session key")
                                .font(.system(size: 15))
                            if !viewModel.timeRemaining.isEmpty {
                                Text(viewModel.timeRemaining)
                                    .font(.caption)
                                    .foregroundStyle(palette.mutedForeground)
                            }
                        }
                        Spacer()
                        Text("Active")
                            .font(.caption.weight(.semibold))
                            .padding(.horizontal, 9)
                            .padding(.vertical, 5)
                            .background(Color.green.opacity(0.12))
                            .foregroundStyle(Color.green)
                            .clipShape(Capsule())
                    }

                    Text("Encrypted server-side for your session. Depending on server configuration, encrypted session data may also be persisted locally to survive restarts. Your raw API key is never stored in the database.")
                        .font(.caption)
                        .foregroundStyle(palette.mutedForeground)

                    sectionTitle("Server", palette: palette)
                    TextField("http://localhost:8000", text: $viewModel.serverURL)
                        .keyboardType(.URL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .font(.system(.footnote, design: .monospaced))
                        .padding(.vertical, 9)
                        .overlay(alignment: .bottom) { palette.border.frame(height: 1) }
                        .onSubmit { viewModel.saveServerURL() }

                    sectionTitle("Account", palette: palette)
                    HStack {
                        Text("Log out")
                            .font(.system(size: 15))
                        Spacer()
                        Button("Log out") { confirmingLogout = true }
                            .buttonStyle(.bordered)
                    }
                }
                .padding(22)
                .frame(maxWidth: 600, alignment: .leading)
                .frame(maxWidth: .infinity)
            }
            .background(palette.background)
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button(action: { dismiss() }) {
                        Image(systemName: "xmark")
                    }
                }
            }
        }
        .onAppear { viewModel.load() }
        .confirmationDialog("Log out?", isPresented: $confirmingLogout) {
            Button("Log out", role: .destructive) {
                viewModel.logout()
                dismiss()
            }
            Button("Cancel", role: .cancel) {}
        }
    }

    private func sectionTitle(_ title: String, palette: ThreadGPTPalette) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.caption.weight(.medium))
                .textCase(.uppercase)
                .tracking(1.5)
                .foregroundStyle(palette.mutedForeground)
            palette.border.frame(height: 1)
        }
    }
}

