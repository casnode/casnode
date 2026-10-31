"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";
import { useFinishLogin } from "@/components/login-options";
import { useSite } from "@/components/site-provider";
import { apiPost, errorMessage } from "@/lib/api";

// Asking for an email with a link to set a new password.
export function ForgotPasswordForm() {
  const { t, site } = useSite();
  const [login, setLogin] = useState("");
  const [sent, setSent] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await apiPost("forgot-password", { login });
      setSent(true);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="login-page">
      <h1>{t("reset.title")}</h1>
      {error ? <div className="alert alert-error">{error}</div> : null}
      {!site.passwordReset ? (
        <div className="alert alert-info">{t("reset.unavailable")}</div>
      ) : sent ? (
        <div className="alert alert-info">{t("reset.sent")}</div>
      ) : (
        <form onSubmit={submit}>
          <p className="instructions mb-4">{t("reset.help")}</p>
          <div className="control-group">
            <label className="form-label" htmlFor="reset-login">
              {t("login.account")}
            </label>
            <input id="reset-login" className="d-input" autoComplete="username" autoFocus value={login} onChange={(e) => setLogin(e.target.value)} />
          </div>
          <div className="login-actions">
            <button type="submit" className="btn btn-primary" disabled={busy || !login.trim()}>
              {t("reset.send")}
            </button>
          </div>
        </form>
      )}
      <p className="login-switch">
        <Link href="/login">{t("reset.back")}</Link>
      </p>
    </div>
  );
}

// Setting a new password through the emailed link.
export function NewPasswordForm({ token }: { token: string }) {
  const { t, settings } = useSite();
  const finishLogin = useFinishLogin();
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const minLength = Number(settings.min_password_length || 8);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (password !== confirm) {
      setError(t("reset.mismatch"));
      return;
    }
    setError("");
    setBusy(true);
    try {
      await apiPost("reset-password", undefined, { token, password });
      await finishLogin();
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };

  return (
    <div className="login-page">
      <h1>{t("reset.newTitle")}</h1>
      {error ? <div className="alert alert-error">{error}</div> : null}
      <form onSubmit={submit}>
        <div className="control-group">
          <label className="form-label" htmlFor="new-password">
            {t("reset.newPassword")}
          </label>
          <input id="new-password" className="d-input" type="password" autoComplete="new-password" autoFocus value={password} onChange={(e) => setPassword(e.target.value)} />
          <div className="instructions">{t("signup.passwordHelp", { count: minLength })}</div>
        </div>
        <div className="control-group">
          <label className="form-label" htmlFor="confirm-password">
            {t("reset.confirm")}
          </label>
          <input id="confirm-password" className="d-input" type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
        </div>
        <div className="login-actions">
          <button type="submit" className="btn btn-primary" disabled={busy || password.length < minLength || !confirm}>
            {t("reset.save")}
          </button>
        </div>
      </form>
    </div>
  );
}
