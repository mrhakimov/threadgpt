import SwiftUI

struct SettingsView: View {
    @StateObject private var viewModel = SettingsViewModel()
    @Environment(\.dismiss) private var dismiss
    @AppStorage("appearance") private var appearance = Appearance.system
    @State private var confirmingLogout = false

    let onLogout: () -> Void

    var body: some View {
        NavigationStack {
            ZStack {
                Color.tgptBackground
                    .ignoresSafeArea()

                List {
                    // Appearance
                    Section {
                        Picker("Theme", selection: $appearance) {
                            ForEach(Appearance.allCases, id: \.self) { option in
                                Text(option.label).tag(option)
                            }
                        }
                        .foregroundColor(.tgptForeground)
                    } header: {
                        Text("Appearance")
                            .foregroundColor(.tgptMutedForeground)
                    }
                    .listRowBackground(Color.tgptCard)

                    // Model
                    Section {
                        if viewModel.isLoadingModels {
                            HStack {
                                Text("Loading models...")
                                    .foregroundColor(.tgptMutedForeground)
                                Spacer()
                                ProgressView()
                            }
                        } else if viewModel.models.isEmpty {
                            Text("No models available")
                                .foregroundColor(.tgptMutedForeground)
                        } else {
                            Picker("Model", selection: Binding(
                                get: { viewModel.selectedModel },
                                set: { viewModel.selectModel($0) }
                            )) {
                                ForEach(viewModel.models, id: \.self) { model in
                                    Text(model).tag(model)
                                }
                            }
                            .foregroundColor(.tgptForeground)
                        }
                    } header: {
                        Text("Model")
                            .foregroundColor(.tgptMutedForeground)
                    }
                    .listRowBackground(Color.tgptCard)

                    // Session
                    Section {
                        HStack {
                            Text("Session expires in")
                                .foregroundColor(.tgptForeground)
                            Spacer()
                            Text(viewModel.timeRemaining)
                                .foregroundColor(.tgptMutedForeground)
                        }
                    } header: {
                        Text("API Key")
                            .foregroundColor(.tgptMutedForeground)
                    }
                    .listRowBackground(Color.tgptCard)

                    // Account
                    Section {
                        Button { confirmingLogout = true } label: {
                            HStack {
                                Image(systemName: "rectangle.portrait.and.arrow.right")
                                Text("Log out")
                            }
                            .foregroundColor(.tgptForeground)
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    } header: {
                        Text("Account")
                            .foregroundColor(.tgptMutedForeground)
                    }
                    .listRowBackground(Color.tgptCard)
                }
                .listStyle(.insetGrouped)
                .scrollContentBackground(.hidden)
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") { dismiss() }
                        .foregroundColor(.tgptForeground)
                }
            }
        }
        .task {
            await viewModel.loadInfo()
            await viewModel.loadModels()
        }
        .onChange(of: appearance) {
            applyAppearance(appearance)
        }
        .confirmationDialog("Log out?", isPresented: $confirmingLogout) {
            Button("Log out", role: .destructive) {
                Task {
                    if await viewModel.logout() {
                        onLogout()
                    }
                }
            }
            Button("Cancel", role: .cancel) {}
        }
    }

    private func applyAppearance(_ appearance: Appearance) {
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene else { return }
        for window in windowScene.windows {
            window.overrideUserInterfaceStyle = appearance.uiStyle
        }
    }
}

enum Appearance: String, CaseIterable {
    case system, light, dark

    var label: String {
        switch self {
        case .system: return "System"
        case .light: return "Light"
        case .dark: return "Dark"
        }
    }

    var uiStyle: UIUserInterfaceStyle {
        switch self {
        case .system: return .unspecified
        case .light: return .light
        case .dark: return .dark
        }
    }
}
