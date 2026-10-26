"use client";

import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost, errorMessage } from "@/lib/api";

interface Setting {
  name: string;
  type: string;
  default: string;
  category: string;
  choices?: string;
  value: string;
}

function SettingRow({ setting, onSaved }: { setting: Setting; onSaved: (value: string) => void }) {
  const { t } = useSite();
  const [value, setValue] = useState(setting.value);
  const dirty = value !== setting.value;

  const save = async (next = value) => {
    try {
      await apiPost("update-site-setting", undefined, { name: setting.name, value: next });
      onSaved(next);
      toast.success(t("adminPage.saved"));
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  let input: React.ReactNode;
  if (setting.type === "bool") {
    input = (
      <label className="form-check">
        <input
          type="checkbox"
          checked={value === "true"}
          onChange={(e) => {
            const next = String(e.target.checked);
            setValue(next);
            save(next);
          }}
        />
        {setting.name.replace(/_/g, " ")}
      </label>
    );
  } else if (setting.type === "enum") {
    input = (
      <select className="d-select max-w-[24em]" value={value} onChange={(e) => setValue(e.target.value)}>
        {(setting.choices ?? "").split("|").map((choice) => (
          <option key={choice} value={choice}>
            {choice}
          </option>
        ))}
      </select>
    );
  } else if (setting.type === "text" || setting.type === "html") {
    input = <textarea className="d-input font-mono text-[0.875em]" rows={setting.type === "text" ? 8 : 4} value={value} onChange={(e) => setValue(e.target.value)} />;
  } else {
    input = <input className="d-input max-w-[32em]" type={setting.type === "integer" ? "number" : "text"} value={value} onChange={(e) => setValue(e.target.value)} />;
  }

  return (
    <div className="setting-row" data-setting={setting.name}>
      <div className="setting-label">
        <h3>{setting.name}</h3>
        <span className="category">{setting.category}</span>
      </div>
      <div className="setting-value">
        {input}
        {dirty && setting.type !== "bool" ? (
          <div className="setting-controls">
            <button type="button" className="btn btn-primary btn-small" onClick={() => save()}>
              <Icon name="check" />
            </button>
            <button type="button" className="btn btn-default btn-small" onClick={() => setValue(setting.value)}>
              <Icon name="xmark" />
            </button>
          </div>
        ) : null}
        {setting.value !== setting.default && setting.type !== "text" ? (
          <button type="button" className="btn-link self-start text-[0.8706em]" onClick={() => {
            setValue(setting.default);
            save(setting.default);
          }}>
            reset ({setting.default || "empty"})
          </button>
        ) : null}
      </div>
    </div>
  );
}

// /admin/site_settings: every setting with a filter box, saved one at a time like Discourse.
export function SiteSettings() {
  const { t, refreshSite } = useSite();
  const [settings, setSettings] = useState<Setting[] | null>(null);
  const [filter, setFilter] = useState("");

  useEffect(() => {
    apiGet<Setting[]>("get-site-settings")
      .then((res) => setSettings(res.data))
      .catch((err) => toast.error(errorMessage(err)));
  }, []);

  const shown = useMemo(() => {
    const q = filter.trim().toLowerCase().replace(/\s+/g, "_");
    return (settings ?? []).filter((s) => !q || s.name.includes(q) || s.category.includes(q));
  }, [settings, filter]);

  return (
    <div>
      <h1 className="page-title">{t("adminPage.settings")}</h1>
      <input className="d-input mb-4 max-w-[30em]" value={filter} placeholder={t("adminPage.filterSettings")} onChange={(e) => setFilter(e.target.value)} />
      {settings === null ? <div className="spinner" /> : null}
      {shown.map((setting) => (
        <SettingRow
          key={setting.name}
          setting={setting}
          onSaved={(value) => {
            setSettings((list) => list?.map((s) => (s.name === setting.name ? { ...s, value } : s)) ?? null);
            refreshSite().catch(() => undefined);
          }}
        />
      ))}
    </div>
  );
}
