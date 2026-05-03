import SwiftUI

struct SettingsView: View {
    @StateObject private var viewModel = SettingsViewModel()
    @Environment(\.dismiss) private var dismiss
    @AppStorage("appearance") private var appearance = Appearance.system

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
                        if viewModel.confirmLogout {
                            HStack {
                                Text("Are you sure?")
                                    .foregroundColor(.tgptForeground)
                                Spacer()
                                Button("Log out") {
                                    Task {
                                        if await viewModel.logout() {
                                            onLogout()
                                        }
                                    }
                                }
                                .foregroundColor(.tgptDestructive)
                                .fontWeight(.semibold)

                                Button("Cancel") {
                                    viewModel.confirmLogout = false
                                }
                                .foregroundColor(.tgptMutedForeground)
                            }
                        } else {
                            Button(action: { viewModel.confirmLogout = true }) {
                                HStack {
                                    Text("Log out")
                                        .foregroundColor(.tgptDestructive)
                                    Spacer()
                                    if viewModel.isLoading {
                                        ProgressView()
                                            .tint(.tgptMutedForeground)
                                    }
                                }
                            }
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
        }
        .onChange(of: appearance) {
            applyAppearance(appearance)
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
