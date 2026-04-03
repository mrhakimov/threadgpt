import { API_URL, JSON_HEADERS, requestVoid } from "@/data/http/client"

export async function authenticate(apiKey: string): Promise<void> {
  await requestVoid(`${API_URL}/api/auth`, {
    method: "POST",
    headers: JSON_HEADERS,
    credentials: "include",
    body: JSON.stringify({ api_key: apiKey }),
  })
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
