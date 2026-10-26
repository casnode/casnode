"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { CategoryBadge } from "@/components/category-badge";
import { CategoryChooser, TagChooser } from "@/components/choosers";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import type { Topic } from "@/lib/types";

// Moving the selected posts to a new topic, or to the end of an existing one; moving all of them merges the topic.
export function MovePostsModal({ topic, postIds, onClose, onMoved }: { topic: Topic; postIds: number[]; onClose: () => void; onMoved: (destination: Topic) => void }) {
  const { t, settings, categories } = useSite();
  const [mode, setMode] = useState<"new" | "existing">("new");
  const [title, setTitle] = useState("");
  const [categoryId, setCategoryId] = useState(topic.categoryId);
  const [tags, setTags] = useState<string[]>([]);
  const [term, setTerm] = useState("");
  const [found, setFound] = useState<Topic[]>([]);
  const [destination, setDestination] = useState(0);
  const [busy, setBusy] = useState(false);
  const isMessage = topic.archetype === "private_message";

  useEffect(() => {
    if (mode !== "existing") {
      return;
    }
    let cancelled = false;
    const timer = setTimeout(() => {
      apiGet<Topic[]>("search-topics-for-move", { term: term.trim(), archetype: topic.archetype })
        .then((res) => {
          if (!cancelled) {
            setFound((res.data ?? []).filter((x) => x.id !== topic.id));
          }
        })
        .catch(() => undefined);
    }, 250);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [mode, term, topic.id, topic.archetype]);

  const submit = async () => {
    setBusy(true);
    try {
      const body = mode === "new" ? { topicId: topic.id, postIds, title, categoryId, tags } : { topicId: topic.id, postIds, destinationTopicId: destination };
      const res = await apiPost<Topic>("move-posts", undefined, body);
      toast.success(t("move.done"));
      onMoved(res.data);
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        title={t("move.title", { count: postIds.length })}
        className="move-posts-modal"
        footer={
          <>
            <button type="button" className="btn btn-primary" disabled={busy || (mode === "new" ? !title.trim() : !destination)} onClick={submit}>
              <Icon name="arrow-right" />
              {mode === "new" ? t("move.toNewTopic") : t("move.toExistingTopic")}
            </button>
            <button type="button" className="btn btn-flat" onClick={onClose}>
              {t("common.cancel")}
            </button>
          </>
        }
      >
        <div className="mb-4 flex gap-2">
          <button type="button" className={`btn btn-small ${mode === "new" ? "btn-primary" : "btn-default"}`} onClick={() => setMode("new")}>
            {t("move.newTopic")}
          </button>
          <button type="button" className={`btn btn-small ${mode === "existing" ? "btn-primary" : "btn-default"}`} onClick={() => setMode("existing")}>
            {t("move.existingTopic")}
          </button>
        </div>
        {mode === "new" ? (
          <>
            <p className="mt-0 text-[var(--primary-high)]">{t("move.newTopicHelp")}</p>
            <div className="control-group">
              <label className="form-label">{t("move.topicTitle")}</label>
              <input className="d-input" value={title} autoFocus onChange={(e) => setTitle(e.target.value)} />
            </div>
            {!isMessage ? (
              <>
                <div className="control-group">
                  <label className="form-label">{t("move.category")}</label>
                  <CategoryChooser value={categoryId} onChange={setCategoryId} placeholder={t("composer.chooseCategory")} />
                </div>
                {settings.tagging_enabled !== "false" ? (
                  <div className="control-group">
                    <label className="form-label">{t("composer.tags")}</label>
                    <TagChooser values={tags} onChange={setTags} placeholder={t("composer.tags")} max={Number(settings.max_tags_per_topic || 5)} />
                  </div>
                ) : null}
              </>
            ) : null}
          </>
        ) : (
          <>
            <p className="mt-0 text-[var(--primary-high)]">{t("move.existingTopicHelp")}</p>
            <input className="d-input mb-2" value={term} autoFocus placeholder={t("move.searchTopics")} onChange={(e) => setTerm(e.target.value)} />
            <div className="move-posts-results">
              {found.map((item) => (
                <label key={item.id} className="flag-action-type">
                  <input type="radio" name="destination" checked={destination === item.id} onChange={() => setDestination(item.id)} />
                  <span className="flag-action-type-details">
                    <strong>{item.title}</strong>
                    {!isMessage ? <CategoryBadge category={categories.find((c) => c.id === item.categoryId)} categories={categories} link={false} /> : null}
                  </span>
                </label>
              ))}
              {found.length === 0 ? <p className="text-[var(--primary-medium)]">{t("move.noTopics")}</p> : null}
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
