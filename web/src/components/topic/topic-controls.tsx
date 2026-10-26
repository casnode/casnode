"use client";

import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { apiPost, errorMessage } from "@/lib/api";
import type { MessageKey } from "@/lib/i18n";
import { NotificationLevel, type Topic, type TopicDetails } from "@/lib/types";
import { isPinned } from "@/lib/utils";

const levels = [
  { level: NotificationLevel.Watching, key: "watching", icon: "bell" },
  { level: NotificationLevel.Tracking, key: "tracking", icon: "circle" },
  { level: NotificationLevel.Regular, key: "normal", icon: "far-bell" },
  { level: NotificationLevel.Muted, key: "muted", icon: "volume-xmark" },
] as const;

// The "Tracking ▾" button of a topic: how closely the user follows it.
export function NotificationsButton({ topic, onChange, showReason }: { topic: Topic; onChange: (level: number) => void; showReason?: boolean }) {
  const { t } = useSite();
  const current = levels.find((l) => l.level === topic.notificationLevel) ?? levels[2];

  const choose = async (level: number) => {
    try {
      await apiPost("set-topic-notification-level", { id: topic.id, level });
      onChange(level);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  return (
    <div className="topic-notifications-button">
      {showReason ? <p className="reason m-0">{t(`notifications.${current.key}.description` as MessageKey)}</p> : null}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button type="button" className="btn btn-default notifications-button">
            <Icon name={current.icon} />
            {t(`notifications.${current.key}` as MessageKey)}
            <Icon name="caret-down" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-[22em]">
          {levels.map((item) => (
            <DropdownMenuItem key={item.level} className={item.level === current.level ? "is-selected" : ""} onSelect={() => choose(item.level)}>
              <Icon name={item.icon} />
              <span className="item-text">
                <span className="item-label">{t(`notifications.${item.key}` as MessageKey)}</span>
                <span className="item-description">{t(`notifications.${item.key}.description` as MessageKey)}</span>
              </span>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

// The wrench menu staff use to close, pin, archive, unlist or delete a topic.
export function TopicAdminMenu({ topic, details, onChange, onSelectPosts }: { topic: Topic; details: TopicDetails; onChange: () => void; onSelectPosts?: () => void }) {
  const { t } = useSite();
  const router = useRouter();

  if (!details.canModerate && !details.canClose) {
    return null;
  }

  const setStatus = async (status: string, enabled: boolean) => {
    try {
      await apiPost("set-topic-status", { id: topic.id, status, enabled: String(enabled) });
      onChange();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const remove = async () => {
    if (!window.confirm(t("admin.confirmDeleteTopic"))) {
      return;
    }
    try {
      await apiPost("delete-topic", { id: topic.id });
      if (details.canModerate) {
        onChange();
      } else {
        router.push("/");
      }
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const recover = async () => {
    try {
      await apiPost("recover-topic", { id: topic.id });
      onChange();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const pinned = isPinned(topic);
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="btn btn-default btn-icon toggle-admin-menu" title={t("admin.menu")}>
          <Icon name="wrench" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={() => setStatus("closed", !topic.closed)}>
          <Icon name={topic.closed ? "circle-check" : "lock"} />
          {topic.closed ? t("admin.open") : t("admin.close")}
        </DropdownMenuItem>
        {details.canModerate ? (
          <>
            {topic.archetype !== "private_message" ? (
              pinned ? (
                <DropdownMenuItem onSelect={() => setStatus("pinned", false)}>
                  <Icon name="thumbtack" />
                  {t("admin.unpin")}
                </DropdownMenuItem>
              ) : (
                <>
                  <DropdownMenuItem onSelect={() => setStatus("pinned", true)}>
                    <Icon name="thumbtack" />
                    {t("admin.pin")}
                  </DropdownMenuItem>
                  <DropdownMenuItem onSelect={() => setStatus("pinned_globally", true)}>
                    <Icon name="thumbtack" />
                    {t("admin.pinGlobally")}
                  </DropdownMenuItem>
                </>
              )
            ) : null}
            <DropdownMenuItem onSelect={() => setStatus("archived", !topic.archived)}>
              <Icon name="box-archive" />
              {topic.archived ? t("admin.unarchive") : t("admin.archive")}
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => setStatus("visible", !topic.visible)}>
              <Icon name={topic.visible ? "eye-slash" : "eye"} />
              {topic.visible ? t("admin.unlist") : t("admin.list")}
            </DropdownMenuItem>
          </>
        ) : null}
        {details.canMovePosts && onSelectPosts ? (
          <DropdownMenuItem onSelect={onSelectPosts}>
            <Icon name="square-check" />
            {t("move.selectPosts")}
          </DropdownMenuItem>
        ) : null}
        {details.canDelete || details.canRecover ? <DropdownMenuSeparator /> : null}
        {details.canDelete && !topic.deleted ? (
          <DropdownMenuItem onSelect={remove} className="text-[var(--danger)]">
            <Icon name="trash-can" />
            {t("admin.deleteTopic")}
          </DropdownMenuItem>
        ) : null}
        {details.canRecover ? (
          <DropdownMenuItem onSelect={recover}>
            <Icon name="rotate-left" />
            {t("admin.recoverTopic")}
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
