"use client";

import { useRouter } from "next/navigation";
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { apiGet } from "@/lib/api";
import { makeT, type Locale, type Translate } from "@/lib/i18n";
import type { Category, SiteInfo, User } from "@/lib/types";

interface SiteContextValue {
  site: SiteInfo;
  currentUser: User | null;
  categories: Category[];
  settings: Record<string, string>;
  locale: Locale;
  t: Translate;
  refreshSite: () => Promise<void>;
  setCounts: (counts: Record<string, number>) => void;
  login: (signup?: boolean) => void;
  casdoorLogin: (signup?: boolean) => void;
  categoryById: (id: number) => Category | undefined;
}

const SiteContext = createContext<SiteContextValue | null>(null);

export function SiteProvider({ initialSite, locale, children }: { initialSite: SiteInfo; locale: Locale; children: ReactNode }) {
  const router = useRouter();
  const [site, setSite] = useState(initialSite);
  const [prevInitial, setPrevInitial] = useState(initialSite);
  // a navigation renders the layout again with fresh data from the server
  if (prevInitial !== initialSite) {
    setPrevInitial(initialSite);
    setSite(initialSite);
  }

  const refreshSite = useCallback(async () => {
    const res = await apiGet<SiteInfo>("get-site");
    setSite(res.data);
  }, []);

  const setCounts = useCallback((counts: Record<string, number>) => {
    setSite((s) => ({ ...s, counts: { ...s.counts, ...counts } }));
  }, []);

  const casdoorLogin = useCallback(
    (signup = false) => {
      const casdoor = site.casdoor;
      const redirectUri = `${window.location.origin}/callback`;
      const params = new URLSearchParams({
        client_id: casdoor.clientId,
        response_type: "code",
        redirect_uri: redirectUri,
        scope: "read",
        state: casdoor.appName,
      });
      // Casdoor is another site, not a page of this one
      // eslint-disable-next-line @next/next/no-location-assign-relative-destination
      window.location.assign(`${casdoor.serverUrl}/${signup ? "signup" : "login"}/oauth/authorize?${params.toString()}`);
    },
    [site.casdoor],
  );

  const login = useCallback(
    (signup = false) => {
      const path = window.location.pathname;
      if (!["/login", "/signup"].includes(path) && !path.startsWith("/password-reset")) {
        try {
          sessionStorage.setItem("casnode-return-to", path + window.location.search);
        } catch {
          // private windows may refuse storage; the user then lands on the home page
        }
      }
      const settings = site.settings ?? {};
      const localLogins = settings.enable_local_logins !== "false";
      const casdoorOnly = !localLogins && !!site.casdoor.serverUrl && !!site.casdoor.clientId && !site.authProviders?.length && !site.devSignin;
      // Casdoor alone needs no page of choices
      if (casdoorOnly) {
        casdoorLogin(signup);
        return;
      }
      router.push(signup && localLogins && settings.allow_new_registrations !== "false" ? "/signup" : "/login");
    },
    [router, site.settings, site.casdoor, site.authProviders, site.devSignin, casdoorLogin],
  );

  const value = useMemo<SiteContextValue>(() => {
    const t = makeT(locale);
    return {
      site,
      currentUser: site.currentUser,
      categories: site.categories ?? [],
      settings: site.settings ?? {},
      locale,
      t,
      refreshSite,
      setCounts,
      login,
      casdoorLogin,
      categoryById: (id: number) => site.categories?.find((c) => c.id === id),
    };
  }, [site, locale, refreshSite, setCounts, login, casdoorLogin]);

  return <SiteContext.Provider value={value}>{children}</SiteContext.Provider>;
}

export function useSite(): SiteContextValue {
  const value = useContext(SiteContext);
  if (!value) {
    throw new Error("useSite() outside SiteProvider");
  }
  return value;
}
