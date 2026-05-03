import Foundation

@MainActor
final class ConversationListViewModel: ObservableObject {
    @Published private(set) var sessions: [ChatSession] = []
    @Published private(set) var hasMore = false
    @Published private(set) var isLoading = false
    @Published private(set) var isLoadingMore = false
    @Published var errorMessage: String?
    @Published var renameTarget: ChatSession?
    @Published var renameText = ""
    @Published var deleteTarget: ChatSession?

    var onRenamed: ((String, String) -> Void)?
    var onDeleted: ((String) -> Void)?

    private let service: SessionService

    init(service: SessionService) {
        self.service = service
    }

    func load(reset: Bool = true) {
        Task {
            if reset {
                isLoading = true
                sessions = []
            }
            errorMessage = nil
            do {
                let page = try await service.listSessions(offset: 0)
                sessions = page.sessions
                hasMore = page.hasMore
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoading = false
        }
    }

    func loadMoreIfNeeded(current session: ChatSession?) {
        guard hasMore, !isLoadingMore, session?.sessionID == sessions.last?.sessionID else { return }
        Task {
            isLoadingMore = true
            do {
                let page = try await service.listSessions(offset: sessions.count)
                sessions.append(contentsOf: page.sessions)
                hasMore = page.hasMore
            } catch {
                errorMessage = error.localizedDescription
            }
            isLoadingMore = false
        }
    }

    func beginRename(_ session: ChatSession) {
        renameTarget = session
        renameText = getSessionLabel(session)
    }

    func commitRename() {
        guard let sessionID = renameTarget?.sessionID else { return }
        let proposedName = renameText
        Task {
            do {
                let name = try await service.renameConversation(sessionID: sessionID, name: proposedName)
                if !name.isEmpty {
                    sessions = sessions.map { session in
                        guard session.sessionID == sessionID else { return session }
                        return ChatSession(
                            sessionID: session.sessionID,
                            assistantID: session.assistantID,
                            systemPrompt: session.systemPrompt,
                            name: name,
                            isNew: session.isNew,
                            createdAt: session.createdAt
                        )
                    }
                    onRenamed?(sessionID, name)
                }
            } catch {
                errorMessage = error.localizedDescription
            }
            renameTarget = nil
            renameText = ""
        }
    }

    func delete(_ session: ChatSession) {
        guard let sessionID = session.sessionID else { return }
        Task {
            do {
                try await service.deleteConversation(sessionID: sessionID)
                sessions.removeAll { $0.sessionID == sessionID }
                onDeleted?(sessionID)
            } catch {
                errorMessage = error.localizedDescription
            }
            deleteTarget = nil
        }
    }
}
