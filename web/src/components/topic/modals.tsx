"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { longDate } from "@/lib/format";
import { PostAction, type Post, type UserMap } from "@/lib/types";

const flagTypes = [
  { type: PostAction.OffTopic, label: "flag.offTopic", description: "flag.offTopic.description" },
  { type: PostAction.Inappropriate, label: "flag.inappropriate", description: "flag.inappropriate.description" },
  { type: PostAction.Spam, label: "flag.spam", description: "flag.spam.description" },
  { type: PostAction.NotifyModerators, label: "flag.other", description: "flag.other.description" },
] as const;

export function FlagModal({ post, onClose, onFlagged }: { post: Post; onClose: () => void; onFlagged: () => void }) {
  const { t } = useSite();
  const [type, setType] = useState<number>(0);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    setBusy(true);
    try {
      await apiPost("flag-post", undefined, { id: post.id, actionType: type, message });
      toast.success(t("flag.thanks"));
      onFlagged();
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        title={t("flag.title")}
        footer={
          <>
            <button type="button" className="btn btn-danger" disabled={!type || busy || (type === PostAction.NotifyModerators && !message.trim())} onClick={submit}>
              <Icon name="flag" />
              {t("flag.submit")}
            </button>
            <button type="button" className="btn btn-flat" onClick={onClose}>
              {t("common.cancel")}
            </button>
          </>
        }
      >
        {flagTypes.map((item) => (
          <label key={item.type} className="flag-action-type">
            <input type="radio" name="flag" checked={type === item.type} onChange={() => setType(item.type)} />
            <span className="flag-action-type-details">
              <strong>{t(item.label)}</strong>
              <span className="description">{t(item.description)}</span>
              {type === item.type && item.type === PostAction.NotifyModerators ? (
                <textarea className="d-input mt-2" rows={3} value={message} placeholder={t("flag.message")} onChange={(e) => setMessage(e.target.value)} />
              ) : null}
            </span>
          </label>
        ))}
      </DialogContent>
    </Dialog>
  );
}

interface Revision {
  id: number;
  number: number;
  userId: number;
  rawBefore: string;
  rawAfter: string;
  reason?: string;
  createdAt: string;
}

export function HistoryModal({ post, onClose }: { post: Post; onClose: () => void }) {
  const { t, locale } = useSite();
  const [revisions, setRevisions] = useState<Revision[] | null>(null);
  const [users, setUsers] = useState<UserMap>({});
  const [index, setIndex] = useState(0);

  useEffect(() => {
    apiGet<Revision[], UserMap>("get-post-revisions", { id: post.id })
      .then((res) => {
        setRevisions(res.data ?? []);
        setUsers(res.data2 ?? {});
        setIndex(Math.max(0, (res.data?.length ?? 1) - 1));
      })
      .catch(() => setRevisions([]));
  }, [post.id]);

  const revision = revisions?.[index];
  const editor = revision ? users[revision.userId] : undefined;

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent title={t("topic.history")} className="w-[min(900px,calc(100vw-2em))]">
        {revisions === null ? <div className="spinner" /> : null}
        {revisions?.length === 0 ? <p>{t("topic.noRevisions")}</p> : null}
        {revision ? (
          <>
            <div className="mb-3 flex flex-wrap items-center gap-2">
              <button type="button" className="btn btn-small" disabled={index === 0} onClick={() => setIndex(index - 1)}>
                <Icon name="chevron-down" className="rotate-90" />
              </button>
              <span className="font-bold">{t("topic.revision", { number: revision.number })}</span>
              <span className="text-[var(--primary-medium)]">/ {post.version}</span>
              <button type="button" className="btn btn-small" disabled={index === revisions!.length - 1} onClick={() => setIndex(index + 1)}>
                <Icon name="chevron-right" />
              </button>
              <span className="ml-auto flex items-center gap-2 text-[var(--primary-medium)]">
                <Avatar user={editor} size={20} />
                {editor?.username} · {longDate(revision.createdAt, locale)}
              </span>
            </div>
            {revision.reason ? <p className="mb-2 text-[var(--primary-high)]">{revision.reason}</p> : null}
            <div className="grid grid-cols-2 gap-3">
              <pre className="max-h-[50vh] overflow-auto whitespace-pre-wrap break-words rounded bg-[var(--danger-low)] p-3 text-[0.875em]">{revision.rawBefore}</pre>
              <pre className="max-h-[50vh] overflow-auto whitespace-pre-wrap break-words rounded bg-[var(--success-low)] p-3 text-[0.875em]">{revision.rawAfter}</pre>
            </div>
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
