"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import type { AuthProvider } from "@/lib/types";

interface ProvidersInfo {
  providers: AuthProvider[];
  callbackUrl: string;
}

const empty: AuthProvider = {
  name: "",
  displayName: "",
  type: "oidc",
  icon: "key",
  clientId: "",
  clientSecret: "",
  issuer: "",
  authUrl: "",
  tokenUrl: "",
  userInfoUrl: "",
  scopes: "",
  idField: "",
  usernameField: "",
  nameField: "",
  emailField: "",
  avatarField: "",
  allowSignup: true,
  enabled: true,
  sortOrder: 0,
};

// The usual providers, filled in; the ones in angle brackets are to be replaced.
const presets: { key: string; label: string; provider: Partial<AuthProvider> }[] = [
  { key: "oidc", label: "OpenID Connect", provider: { type: "oidc", icon: "key" } },
  {
    key: "github",
    label: "GitHub",
    provider: {
      name: "github",
      displayName: "GitHub",
      type: "oauth2",
      icon: "github",
      authUrl: "https://github.com/login/oauth/authorize",
      tokenUrl: "https://github.com/login/oauth/access_token",
      userInfoUrl: "https://api.github.com/user",
      scopes: "read:user user:email",
      idField: "id",
      usernameField: "login",
      nameField: "name",
      emailField: "email",
      avatarField: "avatar_url",
    },
  },
  { key: "google", label: "Google", provider: { name: "google", displayName: "Google", type: "oidc", icon: "google", issuer: "https://accounts.google.com" } },
  {
    key: "microsoft",
    label: "Microsoft Entra ID",
    provider: { name: "microsoft", displayName: "Microsoft", type: "oidc", icon: "microsoft", issuer: "https://login.microsoftonline.com/<tenant id>/v2.0" },
  },
  { key: "gitlab", label: "GitLab", provider: { name: "gitlab", displayName: "GitLab", type: "oidc", icon: "gitlab", issuer: "https://gitlab.com", usernameField: "nickname" } },
  {
    key: "discord",
    label: "Discord",
    provider: {
      name: "discord",
      displayName: "Discord",
      type: "oauth2",
      icon: "discord",
      authUrl: "https://discord.com/oauth2/authorize",
      tokenUrl: "https://discord.com/api/oauth2/token",
      userInfoUrl: "https://discord.com/api/users/@me",
      scopes: "identify email",
      idField: "id",
      usernameField: "username",
      nameField: "global_name",
      emailField: "email",
    },
  },
  { key: "keycloak", label: "Keycloak", provider: { name: "keycloak", displayName: "Keycloak", type: "oidc", icon: "key", issuer: "https://<host>/realms/<realm>" } },
  { key: "casdoor", label: "Casdoor", provider: { name: "sso", displayName: "Casdoor", type: "oidc", icon: "key", issuer: "https://<your casdoor>" } },
  { key: "oauth2", label: "OAuth 2.0", provider: { type: "oauth2", icon: "key", idField: "id" } },
];

const icons = ["key", "github", "google", "microsoft", "gitlab", "discord", "apple", "wechat"];

// /admin/logins: the single sign-on providers members can log in with, besides a password and Casdoor.
export function AdminLogins() {
  const { t, refreshSite } = useSite();
  const [info, setInfo] = useState<ProvidersInfo | null>(null);
  const [editing, setEditing] = useState<{ originalName: string; provider: AuthProvider } | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    apiGet<ProvidersInfo>("get-auth-providers")
      .then((res) => setInfo(res.data))
      .catch((err) => toast.error(errorMessage(err)));
  }, []);
  useEffect(load, [load]);

  const add = (key: string) => {
    const preset = presets.find((item) => item.key === key);
    setEditing({ originalName: "", provider: { ...empty, ...preset?.provider } });
  };

  const save = async () => {
    if (!editing) {
      return;
    }
    setBusy(true);
    try {
      await apiPost("save-auth-provider", undefined, { originalName: editing.originalName, provider: { ...editing.provider, sortOrder: Number(editing.provider.sortOrder) || 0 } });
      toast.success(t("adminPage.saved"));
      setEditing(null);
      load();
      await refreshSite();
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (name: string) => {
    if (!window.confirm(t("adminLogins.confirmDelete", { name }))) {
      return;
    }
    try {
      await apiPost("delete-auth-provider", { name });
      load();
      await refreshSite();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  if (!info) {
    return <div className="spinner" />;
  }

  if (editing) {
    const p = editing.provider;
    const set = (patch: Partial<AuthProvider>) => setEditing({ ...editing, provider: { ...p, ...patch } });
    const text = (field: keyof AuthProvider, label: string, help?: string, type = "text") => (
      <div className="control-group">
        <label className="form-label" htmlFor={`provider-${field}`}>
          {label}
        </label>
        <input
          id={`provider-${field}`}
          className="d-input max-w-[36em]"
          type={type}
          autoComplete="off"
          value={String(p[field] ?? "")}
          disabled={field === "name" && !!editing.originalName}
          onChange={(e) => set({ [field]: e.target.value })}
        />
        {help ? <div className="instructions">{help}</div> : null}
      </div>
    );
    const oauth2 = p.type === "oauth2";
    return (
      <div className="admin-logins max-w-[48em]">
        <h1 className="page-title">{editing.originalName ? t("adminLogins.edit", { name: editing.originalName }) : t("adminLogins.add")}</h1>
        {text("displayName", t("adminLogins.displayName"), t("adminLogins.displayNameHelp"))}
        {text("name", t("adminLogins.name"), t("adminLogins.nameHelp"))}
        <div className="control-group">
          <span className="form-label">{t("adminLogins.callbackUrl")}</span>
          <code className="select-all">{info.callbackUrl + (p.name || "<name>")}</code>
          <div className="instructions">{t("adminLogins.callbackHelp")}</div>
        </div>
        <div className="control-group">
          <label className="form-label" htmlFor="provider-type">
            {t("adminLogins.type")}
          </label>
          <select id="provider-type" className="d-select" value={p.type} onChange={(e) => set({ type: e.target.value as AuthProvider["type"] })}>
            <option value="oidc">OpenID Connect</option>
            <option value="oauth2">OAuth 2.0</option>
          </select>
        </div>
        <div className="control-group">
          <label className="form-label" htmlFor="provider-icon">
            {t("adminLogins.icon")}
          </label>
          <div className="flex items-center gap-2">
            <select id="provider-icon" className="d-select" value={p.icon} onChange={(e) => set({ icon: e.target.value })}>
              {icons.map((icon) => (
                <option key={icon} value={icon}>
                  {icon}
                </option>
              ))}
            </select>
            <Icon name={p.icon || "key"} />
          </div>
        </div>
        {text("clientId", t("adminLogins.clientId"))}
        {text("clientSecret", t("adminLogins.clientSecret"), editing.originalName ? t("adminLogins.secretHelp") : undefined, "password")}
        {oauth2 ? null : text("issuer", t("adminLogins.issuer"), t("adminLogins.issuerHelp"))}
        {text("authUrl", t("adminLogins.authUrl"), oauth2 ? undefined : t("adminLogins.overrideHelp"))}
        {text("tokenUrl", t("adminLogins.tokenUrl"), oauth2 ? undefined : t("adminLogins.overrideHelp"))}
        {text("userInfoUrl", t("adminLogins.userInfoUrl"), oauth2 ? undefined : t("adminLogins.overrideHelp"))}
        {text("scopes", t("adminLogins.scopes"), oauth2 ? undefined : t("adminLogins.scopesHelp"))}
        <h2 className="mt-6 mb-2 text-[1.1487em] font-bold">{t("adminLogins.fields")}</h2>
        <p className="instructions mb-4">{t("adminLogins.fieldsHelp")}</p>
        {text("idField", t("adminLogins.idField"), oauth2 ? undefined : "sub")}
        {text("usernameField", t("adminLogins.usernameField"), oauth2 ? undefined : "preferred_username")}
        {text("nameField", t("adminLogins.nameField"), oauth2 ? undefined : "name")}
        {text("emailField", t("adminLogins.emailField"), oauth2 ? undefined : "email")}
        {text("avatarField", t("adminLogins.avatarField"), oauth2 ? undefined : "picture")}
        <div className="control-group">
          <label className="flex items-center gap-2">
            <input type="checkbox" checked={p.allowSignup} onChange={(e) => set({ allowSignup: e.target.checked })} />
            {t("adminLogins.allowSignup")}
          </label>
          <label className="flex items-center gap-2">
            <input type="checkbox" checked={p.enabled} onChange={(e) => set({ enabled: e.target.checked })} />
            {t("adminLogins.enabled")}
          </label>
        </div>
        {text("sortOrder", t("adminLogins.sortOrder"), undefined, "number")}
        <div className="flex gap-2">
          <button type="button" className="btn btn-primary" disabled={busy} onClick={save}>
            {t("common.save")}
          </button>
          <button type="button" className="btn btn-default" onClick={() => setEditing(null)}>
            {t("common.cancel")}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-[48em]">
      <h1 className="page-title">{t("adminLogins.title")}</h1>
      <p className="text-[var(--primary-high)]">
        {t("adminLogins.help")} <Link href="/admin/site_settings">{t("adminPage.settings")}</Link>
      </p>
      {info.providers.length ? (
        <table className="admin-table mb-6 w-full">
          <thead>
            <tr>
              <th>{t("adminLogins.displayName")}</th>
              <th>{t("adminLogins.name")}</th>
              <th>{t("adminLogins.type")}</th>
              <th>{t("adminLogins.enabled")}</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {info.providers.map((provider) => (
              <tr key={provider.name}>
                <td>
                  <span className="inline-flex items-center gap-2">
                    <Icon name={provider.icon || "key"} />
                    {provider.displayName}
                  </span>
                </td>
                <td>{provider.name}</td>
                <td>{provider.type === "oidc" ? "OpenID Connect" : "OAuth 2.0"}</td>
                <td>{provider.enabled ? t("common.yes") : t("common.no")}</td>
                <td className="text-right">
                  <button type="button" className="btn btn-flat btn-small" onClick={() => setEditing({ originalName: provider.name, provider })}>
                    <Icon name="pencil" />
                    {t("common.edit")}
                  </button>
                  <button type="button" className="btn btn-flat btn-small" onClick={() => remove(provider.name)}>
                    <Icon name="trash-can" />
                    {t("common.delete")}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <div className="alert alert-info">{t("adminLogins.none")}</div>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <span>{t("adminLogins.addPreset")}</span>
        {presets.map((preset) => (
          <button key={preset.key} type="button" className="btn btn-default btn-small" onClick={() => add(preset.key)}>
            {preset.label}
          </button>
        ))}
      </div>
    </div>
  );
}
