import Foundation

final class RemoteChatRepository: ChatRepository {
    private let api = APIClient.shared

    func sendMessage(
        userMessage: String,
        sessionId: String?,
        forceNew: Bool,
        onChunk: @escaping (String) -> Void
    ) async throws -> String? {
        struct Body: Encodable {
            let userMessage: String
            let sessionId: String?
            let forceNew: Bool

            enum CodingKeys: String, CodingKey {
                case userMessage = "user_message"
                case sessionId = "session_id"
                case forceNew = "force_new"
            }
        }

        return try await api.stream(
            method: "POST",
            path: "/api/chat",
            body: Body(userMessage: userMessage, sessionId: sessionId ?? "", forceNew: forceNew),
            onChunk: onChunk
        )
    }

    func fetchHistory(
        sessionId: String?,
        limit: Int,
        offset: Int
    ) async throws -> ConversationHistoryPage {
        var headers: [String: String] = [:]
        if let sessionId {
            headers["X-Session-ID"] = sessionId
        }
        return try await api.request(
            method: "GET",
            path: "/api/history",
            headers: headers,
            queryItems: [
                URLQueryItem(name: "limit", value: "\(limit)"),
                URLQueryItem(name: "offset", value: "\(offset)"),
            ]
        )
    }
}
