import { API_URL, JSON_HEADERS } from "@/data/http/client"

export async function authenticate(apiKey: string): Promise<void> {
  const response = await fetch(`${API_URL}/api/auth`, {
    method: "POST",
    headers: JSON_HEADERS,
    credentials: "include",
    body: JSON.stringify({ api_key: apiKey }),
  })
  if (!response.ok) {
    const text = await response.text().catch(() => "")
    if (response.status === 401) {
      throw new Error("Invalid or expired API key. Please check your key and try again.")
    }
    if (response.status === 502) {
      throw new Error("Could not verify API key with OpenAI. Please try again later.")
    }
    throw new Error(text.trim() || "Authentication failed.")
  }
}

export async function checkAuthentication(): Promise<Response> {
  return fetch(`${API_URL}/api/auth/check`, {
    credentials: "include",
  })
}

export async function logout(): Promise<void> {
  await fetch(`${API_URL}/api/auth/logout`, {
    method: "DELETE",
    credentials: "include",
  })
}

export async function fetchAuthInfo(): Promise<{ expires_at: string }> {
  const res = await fetch(`${API_URL}/api/auth/info`, {
    credentials: "include",
  })
  if (!res.ok) throw new Error("unauthorized")
  return res.json()
}
