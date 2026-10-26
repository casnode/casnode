"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { longAgo } from "@/lib/format";
import type { MessageKey } from "@/lib/i18n";
import type { User } from "@/lib/types";
import { isValidDate, userUrl } from "@/lib/utils";

const filters: { id: string; key: MessageKey }[] = [
  { id: "active", key: "adminPage.active" },
  { id: "new", key: "adminPage.new" },
  { id: "staff", key: "adminPage.staff" },
  { id: "suspended", key: "adminPage.suspended" },
  { id: "silenced", key: "adminPage.silenced" },
];

const farFuture = "9999-12-31T00:00:00Z";
const zero = "0001-01-01T00:00:00Z";

export function AdminUsers() {
  const { t, locale, currentUser } = useSite();
  const [filter, setFilter] = useState("active");
  const [name, setName] = useState("");
  const [users, setUsers] = useState<User[] | null>(null);

  const load = useCallback(async () => {
    const res = await apiGet<User[]>("get-admin-users", { filter, name });
    setUsers(res.data ?? []);
  }, [filter, name]);

  useEffect(() => {
    const timer = setTimeout(() => load().catch(() => setUsers([])), 250);
    return () => clearTimeout(timer);
  }, [load]);

  const update = async (user: User, patch: Partial<User>) => {
    const next = { ...user, ...patch };
    try {
      await apiPost("update-admin-user", undefined, {
        id: next.id,
        admin: next.admin,
        moderator: next.moderator,
        trustLevel: next.trustLevel,
        title: next.title,
        suspendedTill: isValidDate(next.suspendedTill) ? next.suspendedTill : zero,
        suspendReason: next.suspendReason ?? "",
        silencedTill: isValidDate(next.silencedTill) ? next.silencedTill : zero,
      });
      await load();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const suspend = (user: User, field: "suspendedTill" | "silencedTill") => {
    const days = window.prompt(t("adminPage.days"), "7");
    if (!days) {
      return;
    }
    const n = Number(days);
    const till = n > 0 ? new Date(Date.now() + n * 86400000).toISOString() : farFuture;
    const reason = field === "suspendedTill" ? window.prompt(t("adminPage.suspendReason"), "") ?? "" : user.suspendReason;
    update(user, { [field]: till, suspendReason: reason });
  };

  const active = (value?: string) => isValidDate(value) && new Date(value!) > new Date();

  return (
    <div>
      <h1 className="page-title">{t("adminPage.users")}</h1>
      <div className="user-secondary-navigation">
        {filters.map((f) => (
          <button key={f.id} type="button" className={`btn btn-small ${filter === f.id ? "btn-primary" : "btn-flat"}`} onClick={() => setFilter(f.id)}>
            {t(f.key)}
          </button>
        ))}
        <input className="d-input ml-auto max-w-[16em]" value={name} placeholder={t("users.filter")} onChange={(e) => setName(e.target.value)} />
      </div>
      {users === null ? <div className="spinner" /> : null}
      <table className="admin-users-table">
        <thead>
          <tr>
            <th>{t("users.username")}</th>
            <th>{t("user.email")}</th>
            <th>{t("adminPage.lastSeen")}</th>
            <th>{t("adminPage.created")}</th>
            <th>{t("user.trustLevel")}</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {users?.map((user) => (
            <tr key={user.id}>
              <td>
                <Link href={userUrl(user.username)} className="user-info text-[var(--primary)]">
                  <Avatar user={user} size={25} />
                  <span className="font-bold">{user.username}</span>
                  {user.admin ? <Icon name="shield-halved" title={t("user.admin")} className="text-[var(--primary-medium)]" /> : null}
                  {user.moderator ? <Icon name="shield-halved" title={t("user.moderator")} className="text-[var(--tertiary)]" /> : null}
                  {active(user.suspendedTill) ? <Icon name="lock" title={t("adminPage.suspended")} className="text-[var(--danger)]" /> : null}
                  {active(user.silencedTill) ? <Icon name="volume-xmark" title={t("adminPage.silenced")} className="text-[var(--danger)]" /> : null}
                </Link>
              </td>
              <td className="text-[var(--primary-high)]">{user.email}</td>
              <td className="text-[var(--primary-medium)]">{longAgo(user.lastSeenAt, locale)}</td>
              <td className="text-[var(--primary-medium)]">{longAgo(user.createdAt, locale)}</td>
              <td>{user.trustLevel}</td>
              <td className="text-right">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button type="button" className="btn btn-flat btn-icon">
                      <Icon name="ellipsis" />
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    {currentUser?.admin && user.id !== currentUser.id ? (
                      <>
                        <DropdownMenuItem onSelect={() => update(user, { admin: !user.admin })}>{user.admin ? t("adminPage.revokeAdmin") : t("adminPage.grantAdmin")}</DropdownMenuItem>
                        <DropdownMenuItem onSelect={() => update(user, { moderator: !user.moderator })}>{user.moderator ? t("adminPage.revokeModerator") : t("adminPage.grantModerator")}</DropdownMenuItem>
                        <DropdownMenuSeparator />
                      </>
                    ) : null}
                    {active(user.suspendedTill) ? (
                      <DropdownMenuItem onSelect={() => update(user, { suspendedTill: zero, suspendReason: "" })}>{t("adminPage.unsuspend")}</DropdownMenuItem>
                    ) : (
                      <DropdownMenuItem onSelect={() => suspend(user, "suspendedTill")}>{t("adminPage.suspend")}</DropdownMenuItem>
                    )}
                    {active(user.silencedTill) ? (
                      <DropdownMenuItem onSelect={() => update(user, { silencedTill: zero })}>{t("adminPage.unsilence")}</DropdownMenuItem>
                    ) : (
                      <DropdownMenuItem onSelect={() => suspend(user, "silencedTill")}>{t("adminPage.silence")}</DropdownMenuItem>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
