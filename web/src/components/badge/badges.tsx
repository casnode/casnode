"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { BadgeIcon } from "@/components/badge-chip";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { badgeDescription, badgeName, badgeTypeName } from "@/lib/badges";
import { longAgo, longDate } from "@/lib/format";
import type { MessageKey } from "@/lib/i18n";
import type { Badge, BadgeGrant, UserBadgeSummary } from "@/lib/types";
import { badgeUrl, topicUrl, userUrl } from "@/lib/utils";

const groupings = ["getting_started", "community", "posting", "trust_level", "other"];

// A badge as a card: its icon in the color of its type, name, description and how many have it.
export function BadgeCard({ badge, count, grantedAt }: { badge: Badge; count?: number; grantedAt?: string }) {
  const { t, locale } = useSite();
  return (
    <Link href={badgeUrl(badge)} className={`badge-card badge-type-${badgeTypeName(badge.badgeType)}${badge.granted ? " granted" : ""}`}>
      <span className="badge-icon">
        <BadgeIcon badge={badge} />
      </span>
      <span className="badge-info">
        <span className="badge-name">
          {badgeName(badge, t)}
          {count && count > 1 ? <span className="count"> × {count}</span> : null}
          {badge.granted && !grantedAt ? <Icon name="circle-check" className="granted-check" /> : null}
        </span>
        <span className="badge-description">{badgeDescription(badge, t)}</span>
        <span className="badge-granted">{grantedAt ? <span title={longDate(grantedAt, locale)}>{longAgo(grantedAt, locale)}</span> : t("badge.grantCount", { count: badge.grantCount })}</span>
      </span>
    </Link>
  );
}

export function BadgesIndex({ badges }: { badges: Badge[] }) {
  const { t } = useSite();
  return (
    <div className="badges-index">
      <h1 className="page-title">{t("badge.badges")}</h1>
      {groupings.map((grouping) => {
        const list = badges.filter((b) => (b.grouping || "other") === grouping || (grouping === "other" && !groupings.includes(b.grouping)));
        if (list.length === 0) {
          return null;
        }
        return (
          <section key={grouping} className="badge-grouping">
            <h3>{t(`badge.grouping.${grouping}` as MessageKey)}</h3>
            <div className="badge-cards">
              {list.map((badge) => (
                <BadgeCard key={badge.id} badge={badge} />
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}

export function BadgeDetail({ badge, initial, initialMore, username }: { badge: Badge; initial: BadgeGrant[]; initialMore: boolean; username?: string }) {
  const { t, locale, currentUser } = useSite();
  const [grants, setGrants] = useState(initial);
  const [more, setMore] = useState(initialMore);
  const [page, setPage] = useState(0);

  const loadMore = async () => {
    const res = await apiGet<Badge, { grants: BadgeGrant[]; more: boolean }>("get-badge", { id: badge.id, username, page: page + 1 });
    setGrants((list) => [...list, ...res.data2.grants]);
    setMore(res.data2.more);
    setPage(page + 1);
  };

  const revoke = async (grant: BadgeGrant) => {
    if (!window.confirm(t("badge.confirmRevoke", { name: grant.user.username }))) {
      return;
    }
    try {
      await apiPost("revoke-badge", { id: grant.id });
      setGrants((list) => list.filter((g) => g.id !== grant.id));
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  return (
    <div className="badge-detail">
      <div className="badge-detail-header">
        <BadgeCard badge={badge} />
        {badge.longDescription ? <p className="long-description">{badge.longDescription}</p> : null}
      </div>
      {username ? (
        <p>
          {t("badge.grantsOf", { name: username })} <Link href={badgeUrl(badge)}>{t("badge.everyone")}</Link>
        </p>
      ) : null}
      <div className="badge-grants">
        {grants.map((grant) => (
          <div key={grant.id} className="badge-grant">
            <Link href={userUrl(grant.user.username)} data-user-card={grant.user.username}>
              <Avatar user={grant.user} size={45} />
            </Link>
            <div className="badge-grant-info">
              <Link href={userUrl(grant.user.username)} data-user-card={grant.user.username} className="font-bold">
                {grant.user.username}
              </Link>
              <span className="text-[var(--primary-medium)]" title={longDate(grant.grantedAt, locale)}>
                {longAgo(grant.grantedAt, locale)}
              </span>
              {grant.topic && grant.post ? (
                <Link href={topicUrl(grant.topic, grant.post.postNumber)} className="badge-grant-post">
                  {grant.topic.title}
                </Link>
              ) : null}
              {grant.reason ? <span className="text-[var(--primary-high)]">{grant.reason}</span> : null}
            </div>
            {currentUser?.admin ? (
              <button type="button" className="btn btn-flat btn-small ml-auto" title={t("badge.revoke")} onClick={() => revoke(grant)}>
                <Icon name="xmark" />
              </button>
            ) : null}
          </div>
        ))}
      </div>
      {grants.length === 0 ? <p className="text-[var(--primary-medium)]">{t("badge.noGrants")}</p> : null}
      {more ? (
        <button type="button" className="btn btn-default mt-4" onClick={loadMore}>
          {t("common.loadMore")}
        </button>
      ) : null}
    </div>
  );
}

export function UserBadges({ badges }: { badges: UserBadgeSummary[] }) {
  const { t } = useSite();
  if (badges.length === 0) {
    return <p className="text-[var(--primary-medium)]">{t("badge.noBadges")}</p>;
  }
  return (
    <div className="badge-cards">
      {badges.map((item) => (
        <BadgeCard key={item.badge.id} badge={item.badge} count={item.count} grantedAt={item.grantedAt} />
      ))}
    </div>
  );
}

const emptyBadge: Partial<Badge> = { name: "", description: "", longDescription: "", icon: "certificate", imageUrl: "", badgeType: 3, grouping: "other", allowTitle: false, multipleGrant: false, enabled: true };

// /admin/badges: the system badges can be turned off, others created, and any granted by hand.
export function AdminBadges() {
  const { t } = useSite();
  const [badges, setBadges] = useState<Badge[] | null>(null);
  const [editing, setEditing] = useState<Partial<Badge> | null>(null);
  const [grant, setGrant] = useState({ badgeId: 0, username: "", reason: "" });
  const [version, setVersion] = useState(0);

  useEffect(() => {
    apiGet<Badge[]>("get-badges", { all: "true" })
      .then((res) => setBadges(res.data ?? []))
      .catch((err) => toast.error(errorMessage(err)));
  }, [version]);

  const save = async () => {
    if (!editing) {
      return;
    }
    try {
      await apiPost(editing.id ? "update-badge" : "add-badge", undefined, editing);
      toast.success(t("adminPage.saved"));
      setEditing(null);
      setVersion((v) => v + 1);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const toggle = async (badge: Badge) => {
    try {
      await apiPost("update-badge", undefined, { ...badge, enabled: !badge.enabled });
      setVersion((v) => v + 1);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const remove = async (badge: Badge) => {
    if (!window.confirm(t("badge.confirmDelete", { name: badge.name }))) {
      return;
    }
    try {
      await apiPost("delete-badge", { id: badge.id });
      setEditing(null);
      setVersion((v) => v + 1);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const grantBadge = async () => {
    try {
      await apiPost("grant-badge", undefined, { ...grant, username: grant.username.replace(/^@/, "") });
      toast.success(t("badge.granted"));
      setGrant({ ...grant, username: "", reason: "" });
      setVersion((v) => v + 1);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  if (badges === null) {
    return <div className="spinner" />;
  }
  return (
    <div className="admin-badges">
      <h1 className="page-title">{t("badge.badges")}</h1>
      <section className="admin-badges-grant">
        <h3>{t("badge.grant")}</h3>
        <div className="flex flex-wrap items-center gap-2">
          <select className="d-select max-w-[16em]" value={grant.badgeId} onChange={(e) => setGrant({ ...grant, badgeId: Number(e.target.value) })}>
            <option value={0}>{t("badge.chooseBadge")}</option>
            {badges
              .filter((b) => b.enabled)
              .map((b) => (
                <option key={b.id} value={b.id}>
                  {badgeName(b, t)}
                </option>
              ))}
          </select>
          <input className="d-input max-w-[12em]" placeholder={t("group.username")} value={grant.username} onChange={(e) => setGrant({ ...grant, username: e.target.value })} />
          <input className="d-input max-w-[20em]" placeholder={t("badge.reason")} value={grant.reason} onChange={(e) => setGrant({ ...grant, reason: e.target.value })} />
          <button type="button" className="btn btn-primary" disabled={!grant.badgeId || !grant.username.trim()} onClick={grantBadge}>
            {t("badge.grantButton")}
          </button>
        </div>
      </section>
      <div className="my-4">
        <button type="button" className="btn btn-default" onClick={() => setEditing({ ...emptyBadge })}>
          <Icon name="plus" />
          {t("badge.new")}
        </button>
      </div>
      {editing ? (
        <section className="admin-badge-form">
          {!editing.system ? (
            <>
              <div className="control-group">
                <label className="form-label">{t("badge.name")}</label>
                <input className="d-input" value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
              </div>
              <div className="control-group">
                <label className="form-label">{t("badge.description")}</label>
                <input className="d-input" value={editing.description} onChange={(e) => setEditing({ ...editing, description: e.target.value })} />
              </div>
              <div className="control-group">
                <label className="form-label">{t("badge.longDescription")}</label>
                <textarea className="d-input" rows={3} value={editing.longDescription} onChange={(e) => setEditing({ ...editing, longDescription: e.target.value })} />
              </div>
              <div className="control-group">
                <label className="form-label">{t("badge.type")}</label>
                <select className="d-select max-w-[12em]" value={editing.badgeType} onChange={(e) => setEditing({ ...editing, badgeType: Number(e.target.value) })}>
                  <option value={1}>{t("badge.gold")}</option>
                  <option value={2}>{t("badge.silver")}</option>
                  <option value={3}>{t("badge.bronze")}</option>
                </select>
              </div>
              <div className="control-group">
                <label className="form-label">{t("badge.grouping")}</label>
                <select className="d-select max-w-[16em]" value={editing.grouping} onChange={(e) => setEditing({ ...editing, grouping: e.target.value })}>
                  {groupings.map((g) => (
                    <option key={g} value={g}>
                      {t(`badge.grouping.${g}` as MessageKey)}
                    </option>
                  ))}
                </select>
              </div>
            </>
          ) : (
            <h3>{badgeName(editing as Badge, t)}</h3>
          )}
          <div className="control-group">
            <label className="form-label">{t("badge.icon")}</label>
            <input className="d-input max-w-[16em]" value={editing.icon} onChange={(e) => setEditing({ ...editing, icon: e.target.value })} />
          </div>
          <div className="control-group">
            <label className="form-label">{t("badge.imageUrl")}</label>
            <input className="d-input" value={editing.imageUrl} onChange={(e) => setEditing({ ...editing, imageUrl: e.target.value })} />
          </div>
          <div className="control-group">
            <label className="form-check">
              <input type="checkbox" checked={!!editing.allowTitle} onChange={(e) => setEditing({ ...editing, allowTitle: e.target.checked })} />
              {t("badge.allowTitle")}
            </label>
            {!editing.system ? (
              <label className="form-check">
                <input type="checkbox" checked={!!editing.enabled} onChange={(e) => setEditing({ ...editing, enabled: e.target.checked })} />
                {t("badge.enabled")}
              </label>
            ) : null}
          </div>
          <div className="flex gap-2">
            <button type="button" className="btn btn-primary" onClick={save}>
              {t("user.save")}
            </button>
            <button type="button" className="btn btn-flat" onClick={() => setEditing(null)}>
              {t("common.cancel")}
            </button>
            {editing.id && !editing.system ? (
              <button type="button" className="btn btn-danger ml-auto" onClick={() => remove(editing as Badge)}>
                {t("badge.delete")}
              </button>
            ) : null}
          </div>
        </section>
      ) : null}
      <table className="admin-badges-table">
        <tbody>
          {badges.map((badge) => (
            <tr key={badge.id} className={badge.enabled ? "" : "disabled"}>
              <td className={`badge-type-${badgeTypeName(badge.badgeType)}`}>
                <BadgeIcon badge={badge} />
              </td>
              <td>
                <Link href={badgeUrl(badge)} className="font-bold">
                  {badgeName(badge, t)}
                </Link>
                <div className="text-[var(--primary-medium)] text-[0.875em]">{badgeDescription(badge, t)}</div>
              </td>
              <td>{badge.system ? t("badge.system") : t("badge.custom")}</td>
              <td>{t("badge.grantCount", { count: badge.grantCount })}</td>
              <td className="text-right whitespace-nowrap">
                <button type="button" className="btn btn-flat btn-small" onClick={() => toggle(badge)}>
                  {badge.enabled ? t("badge.disable") : t("badge.enable")}
                </button>
                <button type="button" className="btn btn-flat btn-small" onClick={() => setEditing(badge)}>
                  <Icon name="pencil" />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
