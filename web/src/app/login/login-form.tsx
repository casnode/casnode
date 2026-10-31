"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState, type FormEvent } from "react";
import { LoginOptions, useFinishLogin } from "@/components/login-options";
import { useSite } from "@/components/site-provider";
import { apiPost, errorMessage } from "@/lib/api";

// Logging in with a password here, with Casdoor or with a provider on /admin/logins; devSignin lets a developer on
// this computer pick any user name.
export function LoginForm() {
  const { t, site, settings } = useSite();
  const params = useSearchParams();
  const finishLogin = useFinishLogin();
  const [login, setLogin] = useState("");
  const [password, setPassword] = useState("");
  const [devUsername, setDevUsername] = useState("admin");
  const [error, setError] = useState(params.get("error") ?? "");
  const [busy, setBusy] = useState(false);

  const localLogins = settings.enable_local_logins !== "false";
  const allowSignup = localLogins && settings.allow_new_registrations !== "false";
  const sso = (!!site.casdoor.serverUrl && !!site.casdoor.clientId) || (site.authProviders?.length ?? 0) > 0;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await apiPost("login", undefined, { login, password });
      await finishLogin();
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  const devSignin = async () => {
    setError("");
    try {
      await apiPost("dev-signin", { username: devUsername });
      await finishLogin();
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  return (
    <div className="login-page">
      <h1>{t("login.title")}</h1>
      {error ? <div className="alert alert-error">{error}</div> : null}
      {localLogins ? (
        <form onSubmit={submit}>
          <div className="control-group">
            <label className="form-label" htmlFor="login-account">
              {t("login.account")}
            </label>
            <input id="login-account" className="d-input" autoComplete="username" autoFocus value={login} onChange={(e) => setLogin(e.target.value)} />
          </div>
          <div className="control-group">
            <label className="form-label" htmlFor="login-password">
              {t("login.password")}
            </label>
            <input id="login-password" className="d-input" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
          <div className="login-actions">
            <button type="submit" className="btn btn-primary" disabled={busy || !login.trim() || !password}>
              {t("login.submit")}
            </button>
            {site.passwordReset ? <Link href="/password-reset">{t("login.forgot")}</Link> : null}
          </div>
        </form>
      ) : null}
      <LoginOptions divider={localLogins} />
      {allowSignup ? (
        <p className="login-switch">
          {t("login.noAccount")} <Link href="/signup">{t("login.createAccount")}</Link>
        </p>
      ) : null}
      {!localLogins && !sso && !site.devSignin ? <div className="alert alert-info">{t("login.notConfigured")}</div> : null}
      {site.devSignin ? (
        <div className="mt-6">
          <h2 className="mb-2 text-[1.1487em]">{t("login.devTitle")}</h2>
          <p className="mb-3 text-[var(--primary-medium)] text-[0.8706em]">{t("login.devHint")}</p>
          <label className="form-label" htmlFor="dev-username">
            {t("login.username")}
          </label>
          <div className="flex gap-2">
            <input id="dev-username" className="d-input" value={devUsername} onChange={(e) => setDevUsername(e.target.value)} onKeyDown={(e) => e.key === "Enter" && devSignin()} />
            <button type="button" className="btn btn-primary" onClick={devSignin}>
              {t("login.submit")}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
