"use client";

import { useRouter } from "next/navigation";
import { useCallback } from "react";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";

const returnToKey = "casnode-return-to";

function safePath(path: string): string {
  return path.startsWith("/") && !path.startsWith("//") && !path.startsWith("/login") && !path.startsWith("/signup") ? path : "/";
}

// peekReturnTo is the page that sent the user to log in, kept until the login is done.
export function peekReturnTo(): string {
  try {
    return safePath(sessionStorage.getItem(returnToKey) || "/");
  } catch {
    return "/";
  }
}

// useFinishLogin loads the signed-in user and goes back to the page that sent the user to log in.
export function useFinishLogin() {
  const router = useRouter();
  const { refreshSite } = useSite();
  return useCallback(async () => {
    const returnTo = peekReturnTo();
    try {
      sessionStorage.removeItem(returnToKey);
    } catch {
      // nothing kept
    }
    await refreshSite().catch(() => undefined);
    router.replace(returnTo);
    router.refresh();
  }, [router, refreshSite]);
}

// LoginOptions are the buttons for Casdoor and the single sign-on providers set up on /admin/logins.
export function LoginOptions({ signup = false, divider = false }: { signup?: boolean; divider?: boolean }) {
  const { t, site, casdoorLogin } = useSite();
  const casdoor = !!site.casdoor.serverUrl && !!site.casdoor.clientId;
  const providers = site.authProviders ?? [];
  if (!casdoor && providers.length === 0) {
    return null;
  }

  const start = (name: string) => {
    const params = new URLSearchParams({ provider: name, return_to: peekReturnTo() });
    // an API route that redirects to the provider, not a page
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.assign(`/api/auth/start?${params.toString()}`);
  };

  return (
    <div className="login-options">
      {divider ? <div className="login-divider">{t("login.or")}</div> : null}
      {casdoor ? (
        <button type="button" className="btn btn-social" onClick={() => casdoorLogin(signup)}>
          <Icon name="key" />
          {t("login.with", { name: "Casdoor" })}
        </button>
      ) : null}
      {providers.map((provider) => (
        <button key={provider.name} type="button" className={`btn btn-social btn-social-${provider.icon || "key"}`} onClick={() => start(provider.name)}>
          <Icon name={provider.icon || "key"} />
          {t("login.with", { name: provider.displayName })}
        </button>
      ))}
    </div>
  );
}
