"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";
import { LoginOptions, useFinishLogin } from "@/components/login-options";
import { useSite } from "@/components/site-provider";
import { apiPost, errorMessage } from "@/lib/api";

// Creating a local account with a password, or one through Casdoor or a provider on /admin/logins.
export function SignupForm() {
  const { t, settings } = useSite();
  const finishLogin = useFinishLogin();
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const allowSignup = settings.enable_local_logins !== "false" && settings.allow_new_registrations !== "false";
  const minLength = Number(settings.min_password_length || 8);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await apiPost("signup", undefined, { email, username, name, password });
      await finishLogin();
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <div className="login-page">
      <h1>{t("signup.title")}</h1>
      {error ? <div className="alert alert-error">{error}</div> : null}
      {allowSignup ? (
        <form onSubmit={submit}>
          <div className="control-group">
            <label className="form-label" htmlFor="signup-email">
              {t("signup.email")}
            </label>
            <input id="signup-email" className="d-input" type="email" autoComplete="email" autoFocus value={email} onChange={(e) => setEmail(e.target.value)} />
            <div className="instructions">{t("signup.emailHelp")}</div>
          </div>
          <div className="control-group">
            <label className="form-label" htmlFor="signup-username">
              {t("signup.username")}
            </label>
            <input id="signup-username" className="d-input" autoComplete="username" maxLength={20} value={username} onChange={(e) => setUsername(e.target.value)} />
            <div className="instructions">{t("signup.usernameHelp")}</div>
          </div>
          <div className="control-group">
            <label className="form-label" htmlFor="signup-name">
              {t("signup.name")}
            </label>
            <input id="signup-name" className="d-input" autoComplete="name" maxLength={100} value={name} onChange={(e) => setName(e.target.value)} />
            <div className="instructions">{t("signup.nameHelp")}</div>
          </div>
          <div className="control-group">
            <label className="form-label" htmlFor="signup-password">
              {t("signup.password")}
            </label>
            <input id="signup-password" className="d-input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} />
            <div className="instructions">{t("signup.passwordHelp", { count: minLength })}</div>
          </div>
          <div className="login-actions">
            <button type="submit" className="btn btn-primary" disabled={busy || !email.trim() || !username.trim() || password.length < minLength}>
              {t("signup.submit")}
            </button>
          </div>
        </form>
      ) : (
        <div className="alert alert-info">{t("signup.closed")}</div>
      )}
      <LoginOptions signup divider={allowSignup} />
      <p className="login-switch">
        {t("signup.haveAccount")} <Link href="/login">{t("login.title")}</Link>
      </p>
    </div>
  );
}
