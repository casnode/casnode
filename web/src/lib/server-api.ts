import { cookies, headers } from "next/headers";
import { cache } from "react";
import type { ApiResponse, SiteInfo } from "@/lib/types";

// Calls from the Next.js server while rendering a page: straight to the Go backend, passing on the visitor's
// cookies so the page is rendered for whoever is signed in.
const backendUrl = process.env.BACKEND_URL || "http://127.0.0.1:16900";

export async function serverGet<T, T2 = unknown>(path: string, params?: Record<string, string | number | undefined>): Promise<ApiResponse<T, T2>> {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params ?? {})) {
    if (value !== undefined && value !== "") {
      query.set(key, String(value));
    }
  }
  const qs = query.toString();
  const cookieStore = await cookies();
  const headerStore = await headers();
  const forwardedFor = headerStore.get("x-forwarded-for")?.split(",")[0]?.trim() || headerStore.get("x-real-ip") || "";
  try {
    const res = await fetch(`${backendUrl}/api/${path}${qs ? `?${qs}` : ""}`, {
      headers: { cookie: cookieStore.toString(), "X-Visitor-Ip": forwardedFor },
      cache: "no-store",
    });
    return (await res.json()) as ApiResponse<T, T2>;
  } catch (err) {
    return { status: "error", msg: `The backend is not reachable: ${err instanceof Error ? err.message : err}`, data: null as T, data2: null as T2 };
  }
}

export const getSite = cache(async (): Promise<SiteInfo> => {
  const res = await serverGet<SiteInfo>("get-site");
  if (res.status !== "ok") {
    return {
      settings: { title: "Casnode", top_menu: "latest|categories" },
      casdoor: { serverUrl: "", clientId: "", organizationName: "", appName: "" },
      authProviders: [],
      passwordReset: false,
      devSignin: false,
      categories: [],
      topTags: [],
      currentUser: null,
      counts: {},
      defaultLocale: "en",
    };
  }
  return res.data;
});

export async function getOrigin(): Promise<string> {
  const site = await getSite();
  if (site.settings.site_url) {
    return site.settings.site_url.replace(/\/$/, "");
  }
  const headerStore = await headers();
  const host = headerStore.get("host") ?? "localhost";
  const proto = headerStore.get("x-forwarded-proto") ?? (host.startsWith("localhost") || host.startsWith("127.") ? "http" : "https");
  return `${proto}://${host}`;
}
