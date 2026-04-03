export function toErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export function isUnauthorizedError(error: unknown): boolean {
  const message = toErrorMessage(error).toLowerCase()
  return message.includes("unauthorized") || message.includes("401")
}
