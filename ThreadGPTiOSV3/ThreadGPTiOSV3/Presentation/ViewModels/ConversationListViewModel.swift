import Foundation

@MainActor
final class ConversationListViewModel: ObservableObject {
    @Published var sessions: [Session] = []
    @Published var isLoading = false
    @Published var hasMore = false
    @Published var editingId: String?
    @Published var editingName = ""
    @Published var confirmDeleteId: String?

    var onUnauthorized: (() -> Void)?

    private let sessionRepo: SessionRepository

    init(sessionRepo: SessionRepository = AppContainer.shared.sessionRepo) {
        self.sessionRepo = sessionRepo
    }

    func loadSessions() async {
        isLoading = true
        do {
            let page = try await sessionRepo.listSessions(limit: 30, offset: 0)
            sessions = page.sessions
            hasMore = page.hasMore
        } catch let appError as AppError where appError.isUnauthorized {
            onUnauthorized?()
        } catch {}
        isLoading = false
    }

    func loadMore() async {
        guard hasMore, !isLoading else { return }
        isLoading = true
        do {
            let page = try await sessionRepo.listSessions(limit: 30, offset: sessions.count)
            sessions.append(contentsOf: page.sessions)
            hasMore = page.hasMore
        } catch {}
        isLoading = false
    }

    func startRename(_ session: Session) {
        editingId = session.sessionId
        editingName = session.name ?? ""
    }

    func commitRename() async -> String? {
        guard let id = editingId, !editingName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            editingId = nil
            return nil
        }
        let name = editingName.trimmingCharacters(in: .whitespacesAndNewlines)
        do {
            try await sessionRepo.updateSession(id: id, name: name, systemPrompt: nil)
            if let idx = sessions.firstIndex(where: { $0.sessionId == id }) {
                sessions[idx].name = name
            }
            let renamedId = editingId
            editingId = nil
            return renamedId
        } catch {
            editingId = nil
            return nil
        }
    }

    func deleteSession(_ id: String) async {
        do {
            try await sessionRepo.deleteSession(id: id)
            sessions.removeAll { $0.sessionId == id }
            confirmDeleteId = nil
        } catch {}
    }

    func refresh() async {
        await loadSessions()
    }
}
