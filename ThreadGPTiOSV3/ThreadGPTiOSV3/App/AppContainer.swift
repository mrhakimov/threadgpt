import Foundation

final class AppContainer {
    static let shared = AppContainer()

    let authRepo: AuthRepository = RemoteAuthRepository()
    let sessionRepo: SessionRepository = RemoteSessionRepository()
    let chatRepo: ChatRepository = RemoteChatRepository()
    let threadRepo: ThreadRepository = RemoteThreadRepository()

    private init() {}
}
