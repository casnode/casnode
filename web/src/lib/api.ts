import type { ApiResponse } from "@/lib/types";

// Calls from the browser; the pages and the API are on the same origin, so the session cookie goes along.

export class ApiError extends Error {}

export async function apiGet<T, T2 = unknown>(path: string, params?: Record<string, string | number | boolean | undefined>): Promise<ApiResponse<T, T2>> {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params ?? {})) {
    if (value !== undefined && value !== "") {
      query.set(key, String(value));
    }
  }
  const qs = query.toString();
  const res = await fetch(`/api/${path}${qs ? `?${qs}` : ""}`, { credentials: "same-origin" });
  const json = (await res.json()) as ApiResponse<T, T2>;
  if (json.status !== "ok") {
    throw new ApiError(json.msg);
  }
  return json;
}

export async function apiPost<T, T2 = unknown>(
  path: string,
  params?: Record<string, string | number | boolean | undefined>,
  body?: unknown,
): Promise<ApiResponse<T, T2>> {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params ?? {})) {
    if (value !== undefined) {
      query.set(key, String(value));
    }
  }
  const qs = query.toString();
  const init: RequestInit = { method: "POST", credentials: "same-origin" };
  if (body instanceof FormData) {
    init.body = body;
  } else if (body !== undefined) {
    init.body = JSON.stringify(body);
    init.headers = { "Content-Type": "application/json" };
  }
  const res = await fetch(`/api/${path}${qs ? `?${qs}` : ""}`, init);
  const json = (await res.json()) as ApiResponse<T, T2>;
  if (json.status !== "ok") {
    throw new ApiError(json.msg);
  }
  return json;
}

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
