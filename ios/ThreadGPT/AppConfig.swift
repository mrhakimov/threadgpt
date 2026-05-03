import Foundation

enum AppConfig {
    static let defaultBackendURL = "http://localhost:8000"
    static let serverURLDefaultsKey = "threadgpt.serverURL"
    static let messagePageSize = 10
    static let sessionPageSize = 20
    static let initialChatConfirmation = "Context set! Your assistant has been configured with this as its instructions. Send your next message to start chatting."
}
