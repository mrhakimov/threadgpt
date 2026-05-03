import Foundation

struct APIErrorDetail: Codable, Equatable {
    let code: String
    let message: String
    let status: Int?
}

struct APIErrorEnvelope: Codable, Equatable {
    let error: APIErrorDetail
}

enum AppError: LocalizedError, Equatable {
    case invalidURL
    case invalidResponse
    case api(APIErrorDetail)
    case http(status: Int, message: String)
    case unauthorized
    case cancelled
    case unknown(String)

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "The server URL is invalid."
        case .invalidResponse:
            return "The server returned an invalid response."
        case .api(let detail):
            return detail.message
        case .http(_, let message):
            return message
        case .unauthorized:
            return "Please sign in again."
        case .cancelled:
            return "The request was cancelled."
        case .unknown(let message):
            return message
        }
    }

    var isUnauthorized: Bool {
        switch self {
        case .unauthorized:
            return true
        case .api(let detail):
            return detail.status == 401 || detail.status == 403
        case .http(let status, _):
            return status == 401 || status == 403
        default:
            return false
        }
    }
}

extension Error {
    var asAppError: AppError {
        if let appError = self as? AppError {
            return appError
        }
        if (self as NSError).domain == NSURLErrorDomain,
           (self as NSError).code == NSURLErrorCancelled {
            return .cancelled
        }
        return .unknown(localizedDescription)
    }
}

