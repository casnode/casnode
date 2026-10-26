"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { UserChooser } from "@/components/choosers";
import { useComposer } from "@/components/composer-provider";
import { Cooked } from "@/components/cooked";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { longAgo, longDate } from "@/lib/format";
import type { MessageKey } from "@/lib/i18n";
import type { Group, GroupMember, User } from "@/lib/types";
import { groupUrl, userUrl } from "@/lib/utils";

// The groups pages (/g, /g/<name>/...): Discourse's groups, with members, owners, joining and requests.

export function GroupsIndex({ initial }: { initial: Group[] }) {
  const { t, currentUser } = useSite();
  const [term, setTerm] = useState("");
  const [creating, setCreating] = useState(false);
  const router = useRouter();
  const q = term.trim().toLowerCase();
  const groups = initial.filter((g) => !q || g.name.toLowerCase().includes(q) || g.fullName.toLowerCase().includes(q));

  return (
    <div className="groups-index">
      <div className="groups-header">
        <h1 className="page-title m-0">{t("group.groups")}</h1>
        <input className="d-input max-w-[20em]" value={term} placeholder={t("group.filter")} onChange={(e) => setTerm(e.target.value)} />
        {currentUser?.admin ? (
          <button type="button" className="btn btn-primary ml-auto" onClick={() => setCreating(true)}>
            <Icon name="plus" />
            {t("group.new")}
          </button>
        ) : null}
      </div>
      {groups.length === 0 ? <p className="text-[var(--primary-medium)]">{t("group.none")}</p> : null}
      <div className="groups-boxes">
        {groups.map((group) => (
          <Link key={group.id} href={groupUrl(group.name)} className="group-box">
            <div className="group-box-inner">
              <div className="group-info-wrapper">
                <span className="group-avatar-flair">
                  <Icon name={group.automatic ? "shield-halved" : "users"} />
                </span>
                <div className="group-info">
                  <span className="group-user-count">
                    <Icon name="user" />
                    {group.userCount}
                  </span>
                  <h3 className="group-name">{group.fullName || group.name}</h3>
                  {group.fullName ? <div className="user-group-name">{group.name}</div> : null}
                </div>
              </div>
              <div className="group-membership">
                {group.isOwner ? (
                  <span className="is-group-owner">
                    <Icon name="shield-halved" />
                    {t("group.owner")}
                  </span>
                ) : group.isMember ? (
                  <span className="is-group-member">
                    <Icon name="check" />
                    {t("group.member")}
                  </span>
                ) : group.publicJoin ? (
                  <span>{t("group.canJoin")}</span>
                ) : group.requested ? (
                  <span>{t("group.requested")}</span>
                ) : group.allowRequests ? (
                  <span>{t("group.canRequest")}</span>
                ) : group.automatic ? (
                  <span>{t("group.automatic")}</span>
                ) : null}
              </div>
            </div>
          </Link>
        ))}
      </div>
      {creating ? (
        <Dialog open onOpenChange={(open) => !open && setCreating(false)}>
          <DialogContent title={t("group.new")}>
            <GroupForm
              onSaved={(group) => {
                setCreating(false);
                router.push(groupUrl(group.name));
              }}
            />
          </DialogContent>
        </Dialog>
      ) : null}
    </div>
  );
}

const aliasLevels: { value: number; key: MessageKey }[] = [
  { value: 0, key: "group.alias.nobody" },
  { value: 1, key: "group.alias.onlyAdmins" },
  { value: 2, key: "group.alias.modsAndAdmins" },
  { value: 3, key: "group.alias.membersModsAndAdmins" },
  { value: 4, key: "group.alias.ownersModsAndAdmins" },
  { value: 99, key: "group.alias.everyone" },
];

const visibilities: { value: number; key: MessageKey }[] = [
  { value: 0, key: "group.visibility.public" },
  { value: 1, key: "group.visibility.loggedOn" },
  { value: 2, key: "group.visibility.members" },
  { value: 3, key: "group.visibility.staff" },
];

// Creating a group (admins), or changing one (admins and its owners).
export function GroupForm({ group, onSaved }: { group?: Group; onSaved: (group: Group) => void }) {
  const { t, currentUser } = useSite();
  const [form, setForm] = useState<Partial<Group>>(
    group ?? { name: "", fullName: "", bioRaw: "", visibility: 0, mentionableLevel: 3, messageableLevel: 3, publicJoin: false, publicExit: true, allowRequests: false, title: "" },
  );
  const [busy, setBusy] = useState(false);
  const set = (patch: Partial<Group>) => setForm((f) => ({ ...f, ...patch }));
  const admin = !!currentUser?.admin;
  const automatic = !!group?.automatic;

  const save = async () => {
    setBusy(true);
    try {
      const res = await apiPost<Group>(group ? "update-group" : "add-group", undefined, form);
      toast.success(t("user.saved"));
      onSaved(res.data);
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="group-form">
      <div className="control-group">
        <label className="form-label">{t("group.name")}</label>
        <input className="d-input" value={form.name} disabled={!!group && (!admin || automatic)} onChange={(e) => set({ name: e.target.value })} />
        <div className="instructions">{t("group.nameHelp")}</div>
      </div>
      <div className="control-group">
        <label className="form-label">{t("group.fullName")}</label>
        <input className="d-input" value={form.fullName} onChange={(e) => set({ fullName: e.target.value })} />
      </div>
      <div className="control-group">
        <label className="form-label">{t("group.bio")}</label>
        <textarea className="d-input" rows={4} value={form.bioRaw ?? ""} onChange={(e) => set({ bioRaw: e.target.value })} />
      </div>
      {!automatic ? (
        <>
          <div className="control-group">
            <label className="form-label">{t("group.visibility")}</label>
            <select className="d-select" value={form.visibility} disabled={!admin} onChange={(e) => set({ visibility: Number(e.target.value) })}>
              {visibilities.map((v) => (
                <option key={v.value} value={v.value}>
                  {t(v.key)}
                </option>
              ))}
            </select>
          </div>
          <div className="control-group">
            <label className="form-check">
              <input type="checkbox" checked={!!form.publicJoin} onChange={(e) => set({ publicJoin: e.target.checked })} />
              {t("group.publicJoin")}
            </label>
            <label className="form-check">
              <input type="checkbox" checked={!!form.publicExit} onChange={(e) => set({ publicExit: e.target.checked })} />
              {t("group.publicExit")}
            </label>
            <label className="form-check">
              <input type="checkbox" checked={!!form.allowRequests} onChange={(e) => set({ allowRequests: e.target.checked })} />
              {t("group.allowRequests")}
            </label>
          </div>
          <div className="control-group">
            <label className="form-label">{t("group.title")}</label>
            <input className="d-input" value={form.title ?? ""} onChange={(e) => set({ title: e.target.value })} />
            <div className="instructions">{t("group.titleHelp")}</div>
          </div>
        </>
      ) : null}
      <div className="control-group">
        <label className="form-label">{t("group.mentionable")}</label>
        <select className="d-select" value={form.mentionableLevel} onChange={(e) => set({ mentionableLevel: Number(e.target.value) })}>
          {aliasLevels.map((v) => (
            <option key={v.value} value={v.value}>
              {t(v.key)}
            </option>
          ))}
        </select>
      </div>
      <div className="control-group">
        <label className="form-label">{t("group.messageable")}</label>
        <select className="d-select" value={form.messageableLevel} onChange={(e) => set({ messageableLevel: Number(e.target.value) })}>
          {aliasLevels.map((v) => (
            <option key={v.value} value={v.value}>
              {t(v.key)}
            </option>
          ))}
        </select>
      </div>
      <button type="button" className="btn btn-primary" disabled={busy || !form.name?.trim()} onClick={save}>
        {group ? t("user.save") : t("group.create")}
      </button>
    </div>
  );
}

export function GroupHeader({ group }: { group: Group }) {
  const { t, currentUser, login } = useSite();
  const { open: openComposer } = useComposer();
  const router = useRouter();
  const [requesting, setRequesting] = useState(false);
  const [reason, setReason] = useState("");

  const act = async (fn: () => Promise<unknown>, done?: string) => {
    if (!currentUser) {
      login();
      return;
    }
    try {
      await fn();
      if (done) {
        toast.success(done);
      }
      router.refresh();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  return (
    <div className="group-details-container">
      <div className="group-details">
        <span className="group-avatar-flair big">
          <Icon name={group.automatic ? "shield-halved" : "users"} />
        </span>
        <div className="group-info">
          <h1 className="group-info-name">{group.fullName || group.name}</h1>
          {group.fullName ? <div className="group-info-full-name">@{group.name}</div> : null}
          <div className="text-[var(--primary-medium)]">{t("group.memberCount", { count: group.userCount })}</div>
        </div>
        <div className="group-details-button">
          {group.canMessage ? (
            <button type="button" className="btn btn-default" onClick={() => openComposer({ action: "privateMessage", recipients: [group.name] })}>
              <Icon name="envelope" />
              {t("user.message")}
            </button>
          ) : null}
          {group.isMember && group.publicExit && !group.automatic ? (
            <button type="button" className="btn btn-default" onClick={() => act(() => apiPost("remove-group-member", { name: group.name }), t("group.left"))}>
              <Icon name="right-from-bracket" />
              {t("group.leave")}
            </button>
          ) : null}
          {!group.isMember && group.publicJoin && !group.automatic ? (
            <button type="button" className="btn btn-primary" onClick={() => act(() => apiPost("join-group", { name: group.name }), t("group.joined"))}>
              <Icon name="user-plus" />
              {t("group.join")}
            </button>
          ) : null}
          {!group.isMember && !group.publicJoin && group.allowRequests && !group.automatic ? (
            <button type="button" className="btn btn-primary" disabled={group.requested} onClick={() => (currentUser ? setRequesting(true) : login())}>
              <Icon name="user-plus" />
              {group.requested ? t("group.requested") : t("group.request")}
            </button>
          ) : null}
        </div>
      </div>
      {group.bioCooked ? (
        <div className="group-bio">
          <Cooked html={group.bioCooked} />
        </div>
      ) : null}
      {requesting ? (
        <Dialog open onOpenChange={(open) => !open && setRequesting(false)}>
          <DialogContent
            title={t("group.request")}
            footer={
              <button
                type="button"
                className="btn btn-primary"
                onClick={() =>
                  act(async () => {
                    await apiPost("request-group-membership", undefined, { name: group.name, reason });
                    setRequesting(false);
                  }, t("group.requestSent"))
                }
              >
                {t("group.sendRequest")}
              </button>
            }
          >
            <label className="form-label">{t("group.requestReason")}</label>
            <textarea className="d-input" rows={4} value={reason} onChange={(e) => setReason(e.target.value)} />
          </DialogContent>
        </Dialog>
      ) : null}
    </div>
  );
}

export function GroupNav({ group }: { group: Group }) {
  const { t, currentUser } = useSite();
  const pathname = decodeURIComponent(usePathname());
  const base = groupUrl(group.name);
  const tabs: { href: string; key: MessageKey; icon: string; show: boolean; badge?: number }[] = [
    { href: base, key: "group.members", icon: "users", show: true },
    { href: `${base}/activity`, key: "group.activity", icon: "list", show: true },
    { href: `${base}/messages`, key: "group.messages", icon: "envelope", show: !!group.isMember || !!currentUser?.admin },
    { href: `${base}/requests`, key: "group.requests", icon: "user-plus", show: !!group.canManage && group.allowRequests, badge: group.requestCount },
    { href: `${base}/manage`, key: "group.manage", icon: "gear", show: !!group.canManage || (!!currentUser?.admin && group.automatic) },
  ];
  return (
    <nav className="user-navigation">
      <ul className="user-navigation-primary">
        {tabs
          .filter((tab) => tab.show)
          .map((tab) => (
            <li key={tab.href} className="user-nav__item">
              <Link href={tab.href} className={decodeURIComponent(tab.href) === pathname ? "active" : ""}>
                <Icon name={tab.icon} />
                {t(tab.key)}
                {tab.badge ? <span className="badge-notification">{tab.badge}</span> : null}
              </Link>
            </li>
          ))}
      </ul>
    </nav>
  );
}

export function GroupMembers({ group }: { group: Group }) {
  const { t, locale, currentUser } = useSite();
  const [members, setMembers] = useState<GroupMember[] | null>(null);
  const [more, setMore] = useState(false);
  const [page, setPage] = useState(0);
  const [term, setTerm] = useState("");
  const [adding, setAdding] = useState(false);
  const [version, setVersion] = useState(0);

  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(() => {
      apiGet<GroupMember[], boolean>("get-group-members", { name: group.name, term: term.trim(), page: 0 })
        .then((res) => {
          if (!cancelled) {
            setMembers(res.data ?? []);
            setMore(res.data2);
            setPage(0);
          }
        })
        .catch((err) => toast.error(errorMessage(err)));
    }, 200);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [group.name, term, version]);

  const loadMore = async () => {
    const res = await apiGet<GroupMember[], boolean>("get-group-members", { name: group.name, term: term.trim(), page: page + 1 });
    setMembers((list) => [...(list ?? []), ...(res.data ?? [])]);
    setMore(res.data2);
    setPage(page + 1);
  };

  const run = async (fn: () => Promise<unknown>) => {
    try {
      await fn();
      setVersion((v) => v + 1);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  return (
    <div className="group-members">
      <div className="group-members-actions">
        <input className="d-input max-w-[20em]" value={term} placeholder={t("group.filterMembers")} onChange={(e) => setTerm(e.target.value)} />
        {group.canManage ? (
          <button type="button" className="btn btn-primary ml-auto" onClick={() => setAdding(true)}>
            <Icon name="user-plus" />
            {t("group.addMembers")}
          </button>
        ) : null}
      </div>
      {members === null ? <div className="spinner" /> : null}
      {members && members.length === 0 ? <p className="text-[var(--primary-medium)]">{t("group.noMembers")}</p> : null}
      {members && members.length > 0 ? (
        <table className="group-members-table">
          <thead>
            <tr>
              <th>{t("group.username")}</th>
              <th>{t("group.added")}</th>
              {group.canManage ? <th /> : null}
            </tr>
          </thead>
          <tbody>
            {members.map((member) => (
              <tr key={member.user.id}>
                <td>
                  <Link href={userUrl(member.user.username)} data-user-card={member.user.username} className="group-member">
                    <Avatar user={member.user} size={32} />
                    <span>
                      <span className="font-bold">{member.user.username}</span>
                      {member.user.name && member.user.name !== member.user.username ? <span className="ml-2 text-[var(--primary-medium)]">{member.user.name}</span> : null}
                    </span>
                    {member.owner ? (
                      <span className="group-owner-label">
                        <Icon name="shield-halved" />
                        {t("group.owner")}
                      </span>
                    ) : null}
                  </Link>
                </td>
                <td title={longDate(member.addedAt, locale)}>{longAgo(member.addedAt, locale)}</td>
                {group.canManage ? (
                  <td className="text-right">
                    <button
                      type="button"
                      className="btn btn-flat btn-small"
                      onClick={() => run(() => apiPost("set-group-owner", { name: group.name, userId: member.user.id, owner: String(!member.owner) }))}
                    >
                      {member.owner ? t("group.removeOwner") : t("group.makeOwner")}
                    </button>
                    {member.user.id !== currentUser?.id || currentUser?.admin ? (
                      <button
                        type="button"
                        className="btn btn-flat btn-small text-[var(--danger)]"
                        onClick={() => window.confirm(t("group.confirmRemove", { name: member.user.username })) && run(() => apiPost("remove-group-member", { name: group.name, userId: member.user.id }))}
                      >
                        {t("group.remove")}
                      </button>
                    ) : null}
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
      {more ? (
        <button type="button" className="btn btn-default mt-4" onClick={loadMore}>
          {t("common.loadMore")}
        </button>
      ) : null}
      {adding ? (
        <AddMembersModal
          group={group}
          onClose={() => setAdding(false)}
          onAdded={() => {
            setAdding(false);
            setVersion((v) => v + 1);
          }}
        />
      ) : null}
    </div>
  );
}

function AddMembersModal({ group, onClose, onAdded }: { group: Group; onClose: () => void; onAdded: () => void }) {
  const { t } = useSite();
  const [usernames, setUsernames] = useState<string[]>([]);
  const [owner, setOwner] = useState(false);
  const [busy, setBusy] = useState(false);
  const add = async () => {
    setBusy(true);
    try {
      await apiPost("add-group-members", undefined, { name: group.name, usernames, owner });
      onAdded();
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        title={t("group.addMembers")}
        footer={
          <button type="button" className="btn btn-primary" disabled={busy || usernames.length === 0} onClick={add}>
            {t("group.add")}
          </button>
        }
      >
        <UserChooser values={usernames} onChange={setUsernames} placeholder={t("group.usernamesPlaceholder")} />
        <label className="form-check mt-3">
          <input type="checkbox" checked={owner} onChange={(e) => setOwner(e.target.checked)} />
          {t("group.addAsOwners")}
        </label>
      </DialogContent>
    </Dialog>
  );
}

export function GroupRequests({ group }: { group: Group }) {
  const { t, locale } = useSite();
  const router = useRouter();
  const [requests, setRequests] = useState<{ user: User; reason: string; createdAt: string }[] | null>(null);
  const [version, setVersion] = useState(0);

  useEffect(() => {
    apiGet<{ user: User; reason: string; createdAt: string }[]>("get-group-requests", { name: group.name })
      .then((res) => setRequests(res.data ?? []))
      .catch((err) => toast.error(errorMessage(err)));
  }, [group.name, version]);

  const handle = async (userId: number, accept: boolean) => {
    try {
      await apiPost("handle-group-request", { name: group.name, userId, accept: String(accept) });
      setVersion((v) => v + 1);
      router.refresh();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  if (requests === null) {
    return <div className="spinner" />;
  }
  if (requests.length === 0) {
    return <p className="text-[var(--primary-medium)]">{t("group.noRequests")}</p>;
  }
  return (
    <table className="group-members-table">
      <tbody>
        {requests.map((request) => (
          <tr key={request.user.id}>
            <td>
              <Link href={userUrl(request.user.username)} data-user-card={request.user.username} className="group-member">
                <Avatar user={request.user} size={32} />
                <span className="font-bold">{request.user.username}</span>
              </Link>
            </td>
            <td>{request.reason}</td>
            <td title={longDate(request.createdAt, locale)}>{longAgo(request.createdAt, locale)}</td>
            <td className="text-right">
              <button type="button" className="btn btn-primary btn-small" onClick={() => handle(request.user.id, true)}>
                {t("group.accept")}
              </button>
              <button type="button" className="btn btn-flat btn-small" onClick={() => handle(request.user.id, false)}>
                {t("group.deny")}
              </button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export function GroupManage({ group }: { group: Group }) {
  const { t, currentUser } = useSite();
  const router = useRouter();
  const remove = async () => {
    if (!window.confirm(t("group.confirmDelete", { name: group.name }))) {
      return;
    }
    try {
      await apiPost("delete-group", { id: group.id });
      router.push("/g");
      router.refresh();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };
  return (
    <div className="max-w-[40em]">
      <GroupForm
        group={group}
        onSaved={(saved) => {
          router.push(groupUrl(saved.name, "manage"));
          router.refresh();
        }}
      />
      {currentUser?.admin && !group.automatic ? (
        <button type="button" className="btn btn-danger mt-6" onClick={remove}>
          <Icon name="trash-can" />
          {t("group.delete")}
        </button>
      ) : null}
    </div>
  );
}
