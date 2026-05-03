import Foundation

final class RemoteThreadRepository: ThreadRepository {
    private let api = APIClient.shared

    func fetchThread(
        conversationId: String,
        limit: Int,
        offset: Int
    ) async throws -> ThreadPage {
        try await api.request(
            method: "GET",
            path: "/api/thread",
            queryItems: [
                URLQueryItem(name: "conversation_id", value: conversationId),
                URLQueryItem(name: "limit", value: "\(limit)"),
                URLQueryItem(name: "offset", value: "\(offset)"),
            ]
        )
    }

    func sendReply(
        conversationId: String,
        userMessage: String,
        onChunk: @escaping (String) -> Void
    ) async throws {
        struct Body: Encodable {
            let conversationId: String
            let userMessage: String

            enum CodingKeys: String, CodingKey {
                case conversationId = "conversation_id"
                case userMessage = "user_message"
            }
        }

        _ = try await api.stream(
            method: "POST",
            path: "/api/thread",
            body: Body(conversationId: conversationId, userMessage: userMessage),
            onChunk: onChunk
        )
    }
}
