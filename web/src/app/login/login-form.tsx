"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiPost, errorMessage } from "@/lib/api";

// Sign-in goes through Casdoor; without it set up, devSignin lets a developer on this computer pick any user name.
export function LoginForm() {
  const { t, site, login, refreshSite } = useSite();
  const router = useRouter();
  const [username, setUsername] = useState("admin");
  const [error, setError] = useState("");
  const configured = !!site.casdoor.serverUrl && !!site.casdoor.clientId;

  const devSignin = async () => {
    setError("");
    try {
      await apiPost("dev-signin", { username });
      await refreshSite();
      let returnTo = "/";
      try {
        returnTo = sessionStorage.getItem("casnode-return-to") || "/";
      } catch {
        // back to the home page
      }
      router.replace(returnTo.startsWith("/") && returnTo !== "/login" ? returnTo : "/");
      router.refresh();
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  return (
    <div className="login-page">
      <h1>{t("login.title")}</h1>
      {configured ? (
        <button type="button" className="btn btn-primary" onClick={() => login(false)}>
          <Icon name="user" />
          {t("login.submit")}
        </button>
      ) : !site.devSignin ? (
        <div className="alert alert-info">{t("login.notConfigured")}</div>
      ) : null}
      {site.devSignin ? (
        <div className="mt-6">
          <h2 className="mb-2 text-[1.1487em]">{t("login.devTitle")}</h2>
          <p className="mb-3 text-[var(--primary-medium)] text-[0.8706em]">{t("login.devHint")}</p>
          <label className="form-label" htmlFor="dev-username">
            {t("login.username")}
          </label>
          <div className="flex gap-2">
            <input id="dev-username" className="d-input" value={username} onChange={(e) => setUsername(e.target.value)} onKeyDown={(e) => e.key === "Enter" && devSignin()} />
            <button type="button" className="btn btn-primary" onClick={devSignin}>
              {t("login.submit")}
            </button>
          </div>
          {error ? <div className="alert alert-error mt-3">{error}</div> : null}
        </div>
      ) : null}
    </div>
  );
}
