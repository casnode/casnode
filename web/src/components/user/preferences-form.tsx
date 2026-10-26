"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { useSite } from "@/components/site-provider";
import { apiPost, errorMessage } from "@/lib/api";
import { locales } from "@/lib/i18n";
import type { User } from "@/lib/types";

export function PreferencesForm({ user, accountUrl }: { user: User; accountUrl: string }) {
  const { t, currentUser } = useSite();
  const router = useRouter();
  const [bioRaw, setBioRaw] = useState(user.bioRaw ?? "");
  const [location, setLocation] = useState(user.location ?? "");
  const [website, setWebsite] = useState(user.website ?? "");
  const [title, setTitle] = useState(user.title ?? "");
  const [locale, setLocale] = useState(user.locale ?? "");
  const [busy, setBusy] = useState(false);
  const staff = !!currentUser && (currentUser.admin || currentUser.moderator);

  const save = async () => {
    setBusy(true);
    try {
      await apiPost("update-profile", undefined, { username: user.username, bioRaw, location, website, title, locale });
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
      <p className="instructions mb-6">
        {t("user.managedByCasdoor")}{" "}
        {accountUrl ? (
          <a href={accountUrl} target="_blank" rel="noopener">
            {t("user.accountCenter")}
          </a>
        ) : null}
      </p>

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
            Title
          </label>
          <input id="title" className="d-input" value={title} onChange={(e) => setTitle(e.target.value)} />
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
      <button type="button" className="btn btn-primary" disabled={busy} onClick={save}>
        {t("user.save")}
      </button>
    </div>
  );
}
