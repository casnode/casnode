"use client";

import Link from "next/link";
import { useState } from "react";
import { toast } from "sonner";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiPost, errorMessage } from "@/lib/api";

// /admin/email: checking the mail settings by sending a test email.
export function AdminEmail() {
  const { t, currentUser } = useSite();
  const [to, setTo] = useState(currentUser?.email ?? "");
  const [busy, setBusy] = useState(false);

  const send = async () => {
    setBusy(true);
    try {
      await apiPost("send-test-email", { to });
      toast.success(t("adminEmail.sent", { to }));
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="max-w-[40em]">
      <h1 className="page-title">{t("adminEmail.title")}</h1>
      <p className="text-[var(--primary-high)]">
        {t("adminEmail.help")} <Link href="/admin/site_settings">{t("adminPage.settings")}</Link>
      </p>
      <div className="control-group">
        <label className="form-label">{t("adminEmail.to")}</label>
        <div className="flex gap-2">
          <input className="d-input" type="email" value={to} onChange={(e) => setTo(e.target.value)} />
          <button type="button" className="btn btn-primary" disabled={busy || !to.trim()} onClick={send}>
            <Icon name="paper-plane" />
            {t("adminEmail.send")}
          </button>
        </div>
      </div>
    </div>
  );
}
