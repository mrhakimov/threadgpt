export interface ApiErrorPayload {
  code: string
  message: string
  status?: number
}

export class ApiError extends Error {
  readonly code?: string
  readonly status?: number

  constructor(message: string, options: { code?: string; status?: number } = {}) {
    super(message)
    this.name = "ApiError"
    this.code = options.code
    this.status = options.status
  }
}

export function toErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export function isUnauthorizedError(error: unknown): boolean {
  if (error instanceof ApiError) {
    return error.code === "unauthorized"
  }

  const message = toErrorMessage(error).toLowerCase()
  return message.includes("unauthorized") || message.includes("401")
}

export function getApiErrorPayload(value: unknown): ApiErrorPayload | null {
  const raw = isRecord(value) && isRecord(value.error) ? value.error : value
  if (!isRecord(raw) || typeof raw.message !== "string") {
    return null
  }

  return {
    code: typeof raw.code === "string" ? raw.code : "unknown_error",
    message: raw.message,
    status: typeof raw.status === "number" ? raw.status : undefined,
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}
