import Foundation

@MainActor
final class AppContainer: ObservableObject {
    let preferences: AppPreferences
    let authService: AuthService
    let sessionService: SessionService
    let chatService: ChatService
    let threadService: ThreadService

    lazy var rootViewModel = RootViewModel(authService: authService, preferences: preferences)

    init() {
        let preferences = AppPreferences()
        let tokenStore = KeychainTokenStore()
        let apiClient = APIClient(preferences: preferences, tokenStore: tokenStore)
        let authRepository = RemoteAuthRepository(api: apiClient, tokenStore: tokenStore)
        let sessionRepository = RemoteSessionRepository(api: apiClient)
        let chatRepository = RemoteChatRepository(api: apiClient)
        let threadRepository = RemoteThreadRepository(api: apiClient)

        self.preferences = preferences
        self.authService = AuthService(repository: authRepository, preferences: preferences)
        self.sessionService = SessionService(repository: sessionRepository)
        self.chatService = ChatService(sessionRepository: sessionRepository, chatRepository: chatRepository)
        self.threadService = ThreadService(repository: threadRepository)
    }

    func makeLoginViewModel() -> LoginViewModel {
        LoginViewModel(rootViewModel: rootViewModel)
    }

    func makeConversationListViewModel() -> ConversationListViewModel {
        ConversationListViewModel(service: sessionService)
    }

    func makeChatViewModel() -> ChatViewModel {
        ChatViewModel(chatService: chatService, sessionService: sessionService)
    }

    func makeThreadViewModel(parentMessage: ChatMessage, onReplySent: @escaping () -> Void) -> ThreadViewModel {
        ThreadViewModel(parentMessage: parentMessage, service: threadService, onReplySent: onReplySent)
    }

    func makeSettingsViewModel() -> SettingsViewModel {
        SettingsViewModel(authService: authService, rootViewModel: rootViewModel, preferences: preferences)
    }
}

