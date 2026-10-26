import { cookies } from "next/headers";
import { makeT, normalizeLocale, type Locale } from "@/lib/i18n";
import { getSite } from "@/lib/server-api";

// The interface language: the visitor's choice (cookie), then their profile, then the site's default.
export async function getLocale(): Promise<Locale> {
  const cookieStore = await cookies();
  const fromCookie = normalizeLocale(cookieStore.get("locale")?.value);
  if (fromCookie) {
    return fromCookie;
  }
  const site = await getSite();
  return normalizeLocale(site.currentUser?.locale) ?? normalizeLocale(site.defaultLocale) ?? "en";
}

export async function getT() {
  return makeT(await getLocale());
}
