import { authRepository } from "@/repositories/authRepository"
import { storageService } from "@/services/storageService"

export async function authenticateWithApiKey(apiKey: string): Promise<void> {
  await authRepository.authenticate(apiKey)
  storageService.persistAuth()
}

export async function checkAuthorization(): Promise<boolean> {
  const hasCredentials = storageService.hasPersistedAuth()

  try {
    const response = await authRepository.checkAuthentication()

    if (response.status === 401 || response.status === 403) {
      storageService.clearPersistedAuth()
      return false
    }

    if (!response.ok) {
      return hasCredentials
    }

    return true
  } catch {
    return hasCredentials
  }
}

export async function logoutUser(): Promise<void> {
  storageService.clearPersistedAuth()
  await authRepository.logout()
}
