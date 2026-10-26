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

  const login = useCallback(
    (signup = false) => {
      try {
        sessionStorage.setItem("casnode-return-to", window.location.pathname + window.location.search);
      } catch {
        // private windows may refuse storage; the user then lands on the home page
      }
      const casdoor = site.casdoor;
      if (!casdoor.serverUrl || !casdoor.clientId) {
        router.push("/login");
        return;
      }
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
    [router, site.casdoor],
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
      categoryById: (id: number) => site.categories?.find((c) => c.id === id),
    };
  }, [site, locale, refreshSite, setCounts, login]);

  return <SiteContext.Provider value={value}>{children}</SiteContext.Provider>;
}

export function useSite(): SiteContextValue {
  const value = useContext(SiteContext);
  if (!value) {
    throw new Error("useSite() outside SiteProvider");
  }
  return value;
}
