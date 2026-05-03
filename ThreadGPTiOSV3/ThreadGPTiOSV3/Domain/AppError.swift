import Foundation

struct APIErrorPayload: Codable {
    let code: String?
    let message: String?
    let status: Int?
}

struct APIErrorWrapper: Codable {
    let error: APIErrorPayload
}

enum AppError: Error, LocalizedError {
    case unauthorized
    case invalidAPIKey
    case forbidden
    case notFound
    case rateLimited
    case quotaExceeded
    case serverError(String)
    case networkError(String)
    case decodingError(String)

    var errorDescription: String? {
        switch self {
        case .unauthorized:
            return "Your session has expired. Please sign in again."
        case .invalidAPIKey:
            return "Invalid API key. Please check and try again."
        case .forbidden:
            return "You don't have access to this resource."
        case .notFound:
            return "Not found."
        case .rateLimited:
            return "Too many requests. Please wait a moment."
        case .quotaExceeded:
            return "API quota exceeded. Check your OpenAI billing."
        case .serverError(let msg):
            return msg.isEmpty ? "Something went wrong on the server." : msg
        case .networkError(let msg):
            return msg.isEmpty ? "Network error. Check your connection." : msg
        case .decodingError(let msg):
            return "Failed to parse response: \(msg)"
        }
    }

    var isUnauthorized: Bool {
        if case .unauthorized = self { return true }
        if case .invalidAPIKey = self { return true }
        return false
    }

    static func from(payload: APIErrorPayload) -> AppError {
        switch payload.code {
        case "unauthorized":
            return .unauthorized
        case "invalid_api_key":
            return .invalidAPIKey
        case "forbidden":
            return .forbidden
        case "not_found":
            return .notFound
        case "rate_limited":
            return .rateLimited
        case "quota_exceeded":
            return .quotaExceeded
        default:
            return .serverError(payload.message ?? "Unknown error")
        }
    }

    static func from(statusCode: Int, data: Data?) -> AppError {
        if let data, let wrapper = try? JSONDecoder().decode(APIErrorWrapper.self, from: data) {
            return from(payload: wrapper.error)
        }
        switch statusCode {
        case 401, 403: return .unauthorized
        case 404: return .notFound
        case 429: return .rateLimited
        default: return .serverError("Server error (\(statusCode))")
        }
    }
}
