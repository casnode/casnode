"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { locales } from "@/lib/i18n";
import type { User } from "@/lib/types";

interface Preferences {
  emailLevel: string;
  emailMessagesLevel: string;
  emailEnabled: boolean;
  titles: string[] | null;
}

export function PreferencesForm({ user, accountUrl }: { user: User; accountUrl: string }) {
  const { t, currentUser } = useSite();
  const router = useRouter();
  const [bioRaw, setBioRaw] = useState(user.bioRaw ?? "");
  const [location, setLocation] = useState(user.location ?? "");
  const [website, setWebsite] = useState(user.website ?? "");
  const [title, setTitle] = useState(user.title ?? "");
  const [locale, setLocale] = useState(user.locale ?? "");
  const [busy, setBusy] = useState(false);
  const [prefs, setPrefs] = useState<Preferences | null>(null);
  const [emailLevel, setEmailLevel] = useState("");
  const [emailMessagesLevel, setEmailMessagesLevel] = useState("");

  useEffect(() => {
    apiGet<Preferences>("get-user-preferences", { username: user.username })
      .then((res) => {
        setPrefs(res.data);
        setEmailLevel(res.data.emailLevel);
        setEmailMessagesLevel(res.data.emailMessagesLevel);
      })
      .catch(() => undefined);
  }, [user.username]);

  const staff = !!currentUser && (currentUser.admin || currentUser.moderator);

  const save = async () => {
    setBusy(true);
    try {
      await apiPost("update-profile", undefined, { username: user.username, bioRaw, location, website, title, locale, emailLevel, emailMessagesLevel });
      if (currentUser?.id === user.id) {
        document.cookie = locale ? `locale=${locale}; path=/; max-age=31536000; samesite=lax` : "locale=; path=/; max-age=0";
      }
      toast.success(t("user.saved"));
      router.refresh();
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="preferences-form">
      <h2 className="mb-4 text-[1.3195em] font-bold">{t("user.account")}</h2>
      <div className="control-group">
        <span className="control-label">{t("user.username")}</span>
        <div>{user.username}</div>
      </div>
      {user.email ? (
        <div className="control-group">
          <span className="control-label">{t("user.email")}</span>
          <div>{user.email}</div>
        </div>
      ) : null}
      {user.loginType === "local" ? (
        currentUser?.id === user.id ? (
          <ChangePassword />
        ) : null
      ) : !user.loginType ? (
        <p className="instructions mb-6">
          {t("user.managedByCasdoor")}{" "}
          {accountUrl ? (
            <a href={accountUrl} target="_blank" rel="noopener">
              {t("user.accountCenter")}
            </a>
          ) : null}
        </p>
      ) : null}

      <h2 className="mb-4 text-[1.3195em] font-bold">{t("user.profile")}</h2>
      <div className="control-group">
        <label className="control-label" htmlFor="bio">
          {t("user.aboutMe")}
        </label>
        <textarea id="bio" className="d-input" rows={6} value={bioRaw} onChange={(e) => setBioRaw(e.target.value)} />
      </div>
      <div className="control-group">
        <label className="control-label" htmlFor="location">
          {t("user.location")}
        </label>
        <input id="location" className="d-input" value={location} onChange={(e) => setLocation(e.target.value)} />
      </div>
      <div className="control-group">
        <label className="control-label" htmlFor="website">
          {t("user.website")}
        </label>
        <input id="website" className="d-input" value={website} onChange={(e) => setWebsite(e.target.value)} />
      </div>
      {staff ? (
        <div className="control-group">
          <label className="control-label" htmlFor="title">
            {t("user.title")}
          </label>
          <input id="title" className="d-input" value={title} onChange={(e) => setTitle(e.target.value)} />
        </div>
      ) : prefs?.titles?.length ? (
        <div className="control-group">
          <label className="control-label" htmlFor="title">
            {t("user.title")}
          </label>
          <select id="title" className="d-select" value={title} onChange={(e) => setTitle(e.target.value)}>
            <option value="">{t("user.noTitle")}</option>
            {prefs.titles.map((item) => (
              <option key={item} value={item}>
                {item}
              </option>
            ))}
          </select>
          <div className="instructions">{t("user.titleHelp")}</div>
        </div>
      ) : null}
      <div className="control-group">
        <label className="control-label" htmlFor="locale">
          {t("user.interfaceLanguage")}
        </label>
        <select id="locale" className="d-select" value={locale} onChange={(e) => setLocale(e.target.value)}>
          <option value="">—</option>
          {locales.map((l) => (
            <option key={l.value} value={l.value}>
              {l.label}
            </option>
          ))}
        </select>
      </div>
      {prefs ? (
        <>
          <h2 className="mt-6 mb-4 text-[1.3195em] font-bold" id="emails">
            {t("user.emails")}
          </h2>
          {!prefs.emailEnabled ? <p className="instructions">{t("user.emailOff")}</p> : null}
          <div className="control-group">
            <label className="control-label" htmlFor="email-level">
              {t("user.emailLevel")}
            </label>
            <select id="email-level" className="d-select" value={emailLevel} onChange={(e) => setEmailLevel(e.target.value)}>
              <option value="always">{t("user.emailAlways")}</option>
              <option value="away">{t("user.emailAway")}</option>
              <option value="never">{t("user.emailNever")}</option>
            </select>
          </div>
          <div className="control-group">
            <label className="control-label" htmlFor="email-messages-level">
              {t("user.emailMessagesLevel")}
            </label>
            <select id="email-messages-level" className="d-select" value={emailMessagesLevel} onChange={(e) => setEmailMessagesLevel(e.target.value)}>
              <option value="always">{t("user.emailAlways")}</option>
              <option value="away">{t("user.emailAway")}</option>
              <option value="never">{t("user.emailNever")}</option>
            </select>
          </div>
        </>
      ) : null}
      <button type="button" className="btn btn-primary" disabled={busy} onClick={save}>
        {t("user.save")}
      </button>
    </div>
  );
}

// ChangePassword is for local accounts, which have a password here.
function ChangePassword() {
  const { t, settings } = useSite();
  const [open, setOpen] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const minLength = Number(settings.min_password_length || 8);

  const save = async () => {
    setBusy(true);
    try {
      await apiPost("change-password", undefined, { currentPassword, newPassword });
      toast.success(t("user.passwordChanged"));
      setOpen(false);
      setCurrentPassword("");
      setNewPassword("");
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="control-group mb-6">
      <span className="control-label">{t("login.password")}</span>
      {open ? (
        <div className="flex max-w-[24em] flex-col gap-2">
          <input className="d-input" type="password" autoComplete="current-password" placeholder={t("user.currentPassword")} value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />
          <input className="d-input" type="password" autoComplete="new-password" placeholder={t("reset.newPassword")} value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
          <div className="instructions">{t("signup.passwordHelp", { count: minLength })}</div>
          <div className="flex gap-2">
            <button type="button" className="btn btn-primary" disabled={busy || !currentPassword || newPassword.length < minLength} onClick={save}>
              {t("reset.save")}
            </button>
            <button type="button" className="btn btn-default" onClick={() => setOpen(false)}>
              {t("common.cancel")}
            </button>
          </div>
        </div>
      ) : (
        <button type="button" className="btn btn-default btn-small" onClick={() => setOpen(true)}>
          {t("user.changePassword")}
        </button>
      )}
    </div>
  );
}
