import type { Session } from "./types";
let token = "";
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  const response = await fetch(`/api${path}`, {
    method,
    headers:
      method === "GET"
        ? {}
        : { "Content-Type": "application/json", "X-DevHub-Token": token },
    body: method === "GET" ? undefined : JSON.stringify(body ?? {}),
  });
  if (!response.ok) {
    const data = await response
      .json()
      .catch(() => ({ error: `HTTP ${response.status}` }));
    throw new Error(data.error || `HTTP ${response.status}`);
  }
  return response.json() as Promise<T>;
}
export async function session() {
  const value = await api<Session>("/session");
  token = value.token;
  return value;
}
