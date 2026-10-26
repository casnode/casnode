"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { CategoryChooser, groupsAsUsers, TagChooser, UserChooser } from "@/components/choosers";
import { postSavedEvent, useComposer, type PostSavedDetail } from "@/components/composer-provider";
import { Cooked } from "@/components/cooked";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { PollBuilder } from "@/components/poll-builder";
import type { Group, Post, Topic, Upload, User } from "@/lib/types";
import { topicUrl, userUrl } from "@/lib/utils";

interface Draft {
  title: string;
  raw: string;
  categoryId: number;
  tags: string[];
  recipients: string[];
}

const minHeight = 200;

export function Composer() {
  const { composer, close, setMinimized, pendingInsert, consumeInsert } = useComposer();
  const { t, settings, categories, currentUser, refreshSite } = useSite();
  const router = useRouter();
  const pathname = usePathname();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  const [title, setTitle] = useState("");
  const [raw, setRaw] = useState("");
  const [categoryId, setCategoryId] = useState(0);
  const [tags, setTags] = useState<string[]>([]);
  const [recipients, setRecipients] = useState<string[]>([]);
  const [editReason, setEditReason] = useState("");
  const [preview, setPreview] = useState("");
  const [showPreview, setShowPreview] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [uploading, setUploading] = useState(0);
  const [error, setError] = useState("");
  const [height, setHeight] = useState(400);
  const [mention, setMention] = useState<{ term: string; start: number; users: User[]; index: number } | null>(null);
  const [loadedKey, setLoadedKey] = useState<number | null>(null);
  const [pollBuilder, setPollBuilder] = useState(false);

  const action = composer?.action;
  const isFirstPostEdit = action === "edit" && composer?.post?.postNumber === 1;
  const showTitle = action === "createTopic" || action === "privateMessage" || isFirstPostEdit;
  const showCategory = (action === "createTopic" || (isFirstPostEdit && composer?.topic?.archetype !== "private_message")) && categories.length > 0;
  const draftKey = composer ? `casnode-draft-${composer.action}-${composer.topic?.id ?? "new"}-${composer.post?.id ?? composer.replyToPost?.postNumber ?? ""}` : "";

  // a new composer starts from the options, or from the draft saved for the same thing
  if (composer && loadedKey !== composer.key) {
    setLoadedKey(composer.key);
    let draft: Draft | null = null;
    if (composer.action !== "edit" && !composer.raw) {
      try {
        draft = JSON.parse(localStorage.getItem(draftKey) ?? "null");
      } catch {
        draft = null;
      }
    }
    setTitle(draft?.title ?? composer.title ?? composer.topic?.title ?? "");
    setRaw(draft?.raw ?? composer.raw ?? composer.post?.raw ?? "");
    setCategoryId(draft?.categoryId ?? composer.categoryId ?? composer.topic?.categoryId ?? 0);
    setTags(draft?.tags ?? composer.topic?.tags ?? []);
    setRecipients(draft?.recipients ?? composer.recipients ?? []);
    setEditReason("");
    setError("");
    setPreview("");
  }

  useEffect(() => {
    if (composer && !composer.minimized) {
      const timer = setTimeout(() => {
        const el = textareaRef.current;
        if (el && !showTitle) {
          el.focus();
          el.setSelectionRange(el.value.length, el.value.length);
        }
      }, 50);
      return () => clearTimeout(timer);
    }
  }, [composer?.key, composer?.minimized, composer, showTitle]);

  // the page scrolls past its end by the composer's height, so nothing hides behind it
  useEffect(() => {
    document.body.style.paddingBottom = composer ? `${composer.minimized ? 40 : height}px` : "";
    return () => {
      document.body.style.paddingBottom = "";
    };
  }, [composer, height]);

  useEffect(() => {
    if (!composer || composer.action === "edit") {
      return;
    }
    const timer = setTimeout(() => {
      try {
        if (raw || title) {
          localStorage.setItem(draftKey, JSON.stringify({ title, raw, categoryId, tags, recipients } satisfies Draft));
        }
      } catch {
        // drafts are just not kept
      }
    }, 500);
    return () => clearTimeout(timer);
  }, [composer, draftKey, title, raw, categoryId, tags, recipients]);

  useEffect(() => {
    if (!composer || !showPreview) {
      return;
    }
    if (!raw.trim()) {
      return;
    }
    const timer = setTimeout(async () => {
      try {
        const res = await apiPost<string>("preview", undefined, { raw });
        setPreview(res.data);
      } catch {
        // the preview stays as it was
      }
    }, 350);
    return () => clearTimeout(timer);
  }, [raw, composer, showPreview]);

  const insertAtCaret = useCallback((text: string, selectInserted = false) => {
    const el = textareaRef.current;
    if (!el) {
      setRaw((r) => r + text);
      return;
    }
    const start = el.selectionStart;
    const end = el.selectionEnd;
    const value = el.value;
    const next = value.slice(0, start) + text + value.slice(end);
    setRaw(next);
    requestAnimationFrame(() => {
      el.focus();
      if (selectInserted) {
        el.setSelectionRange(start, start + text.length);
      } else {
        el.setSelectionRange(start + text.length, start + text.length);
      }
    });
  }, []);

  useEffect(() => {
    if (pendingInsert && composer) {
      const text = pendingInsert;
      consumeInsert();
      // wait for the textarea of a composer that has just been opened
      setTimeout(() => {
        const el = textareaRef.current;
        const prefix = el && el.value && !el.value.endsWith("\n\n") ? (el.value.endsWith("\n") ? "\n" : "\n\n") : "";
        if (el) {
          el.setSelectionRange(el.value.length, el.value.length);
        }
        insertAtCaret(prefix + text);
      }, 80);
    }
  }, [pendingInsert, composer, consumeInsert, insertAtCaret]);

  const wrap = (before: string, after: string, placeholder: string) => {
    const el = textareaRef.current;
    if (!el) {
      return;
    }
    const start = el.selectionStart;
    const end = el.selectionEnd;
    const selected = el.value.slice(start, end) || placeholder;
    const next = el.value.slice(0, start) + before + selected + after + el.value.slice(end);
    setRaw(next);
    requestAnimationFrame(() => {
      el.focus();
      el.setSelectionRange(start + before.length, start + before.length + selected.length);
    });
  };

  const prefixLines = (prefix: (i: number) => string, placeholder: string) => {
    const el = textareaRef.current;
    if (!el) {
      return;
    }
    const value = el.value;
    let start = el.selectionStart;
    const end = el.selectionEnd;
    start = value.lastIndexOf("\n", start - 1) + 1;
    const block = value.slice(start, end) || placeholder;
    const lines = block.split("\n").map((line, i) => prefix(i) + line);
    const text = lines.join("\n");
    const lead = start > 0 && value[start - 1] !== "\n" ? "\n" : "";
    setRaw(value.slice(0, start) + lead + text + value.slice(end));
    requestAnimationFrame(() => {
      el.focus();
      el.setSelectionRange(start + lead.length, start + lead.length + text.length);
    });
  };

  const uploadFiles = async (files: FileList | File[]) => {
    for (const file of Array.from(files)) {
      setUploading((n) => n + 1);
      const placeholder = `[${t("composer.uploading")} ${file.name}]()`;
      insertAtCaret(placeholder + "\n");
      try {
        const form = new FormData();
        form.append("file", file);
        const res = await apiPost<Upload>("upload", undefined, form);
        const upload = res.data;
        const isImage = /^(jpe?g|png|gif|webp|svg)$/.test(upload.extension);
        const markdown = isImage ? `![${upload.originalFilename}](${upload.url})` : `[${upload.originalFilename}](${upload.url})`;
        setRaw((r) => r.replace(placeholder, markdown));
      } catch (err) {
        setRaw((r) => r.replace(placeholder + "\n", ""));
        toast.error(errorMessage(err));
      } finally {
        setUploading((n) => n - 1);
      }
    }
  };

  // @mentions: the user names that start like what is typed after "@"
  const updateMention = async (value: string, caret: number) => {
    const before = value.slice(0, caret);
    const match = /(^|[\s(])@([\w.\-]{0,30})$/.exec(before);
    if (!match) {
      setMention(null);
      return;
    }
    const term = match[2];
    const start = caret - term.length - 1;
    try {
      const res = await apiGet<User[], Group[] | null>("search-users", { term, groups: "mention" });
      const users = [...(res.data ?? []), ...groupsAsUsers(res.data2)];
      setMention(users.length > 0 ? { term, start, users, index: 0 } : null);
    } catch {
      setMention(null);
    }
  };

  const chooseMention = (user: User) => {
    if (!mention) {
      return;
    }
    const el = textareaRef.current;
    const value = el?.value ?? raw;
    const caret = mention.start + 1 + mention.term.length;
    const next = value.slice(0, mention.start) + "@" + user.username + " " + value.slice(caret);
    setRaw(next);
    setMention(null);
    requestAnimationFrame(() => {
      const pos = mention.start + user.username.length + 2;
      el?.focus();
      el?.setSelectionRange(pos, pos);
    });
  };

  const discard = () => {
    if ((raw.trim() || title.trim()) && composer?.action !== "edit" && !window.confirm(t("composer.confirmDiscard"))) {
      return;
    }
    try {
      localStorage.removeItem(draftKey);
    } catch {
      // nothing was stored
    }
    close();
  };

  const submit = async () => {
    if (!composer || submitting) {
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      if (composer.action === "createTopic" || composer.action === "privateMessage") {
        const res = await apiPost<Topic>("add-topic", undefined, {
          title,
          raw,
          categoryId,
          tags,
          archetype: composer.action === "privateMessage" ? "private_message" : "regular",
          recipients,
        });
        localStorage.removeItem(draftKey);
        close();
        router.push(topicUrl(res.data));
      } else if (composer.action === "reply" && composer.topic) {
        const res = await apiPost<Post>("add-post", undefined, { topicId: composer.topic.id, raw, replyToPostNumber: composer.replyToPost?.postNumber ?? 0 });
        localStorage.removeItem(draftKey);
        close();
        const url = topicUrl(composer.topic);
        if (pathname.startsWith(url)) {
          window.dispatchEvent(new CustomEvent<PostSavedDetail>(postSavedEvent, { detail: { topicId: composer.topic.id, post: res.data, edited: false } }));
        } else {
          router.push(topicUrl(composer.topic, res.data.postNumber));
        }
      } else if (composer.action === "edit" && composer.post) {
        if (isFirstPostEdit && composer.topic && (title !== composer.topic.title || categoryId !== composer.topic.categoryId || tags.join() !== (composer.topic.tags ?? []).join())) {
          await apiPost("update-topic", undefined, { id: composer.topic.id, title, categoryId, tags });
        }
        const res = await apiPost<Post>("update-post", undefined, { id: composer.post.id, raw, editReason });
        close();
        window.dispatchEvent(new CustomEvent<PostSavedDetail>(postSavedEvent, { detail: { topicId: composer.post.topicId, post: res.data, edited: true } }));
        router.refresh();
      }
      refreshSite().catch(() => undefined);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSubmitting(false);
    }
  };

  const startResize = (e: React.PointerEvent) => {
    e.preventDefault();
    const startY = e.clientY;
    const startHeight = height;
    const move = (ev: PointerEvent) => {
      const next = Math.min(window.innerHeight - 60, Math.max(minHeight, startHeight + (startY - ev.clientY)));
      setHeight(next);
    };
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  };

  if (!composer || !currentUser) {
    return null;
  }

  if (composer.minimized) {
    return (
      <div id="reply-control" className="draft" style={{ height: 40 }}>
        <div className="draft-title" onClick={() => setMinimized(false)}>
          <Icon name="pencil" />
          <span className="truncate">{title || composer.topic?.title || t("composer.draft")}</span>
          <span className="ml-auto flex gap-1">
            <button type="button" className="btn btn-flat btn-icon" title={t("composer.toggle")} onClick={() => setMinimized(false)}>
              <Icon name="angles-up" />
            </button>
            <button type="button" className="btn btn-flat btn-icon" title={t("composer.discard")} onClick={(e) => {
              e.stopPropagation();
              discard();
            }}>
              <Icon name="xmark" />
            </button>
          </span>
        </div>
      </div>
    );
  }

  const header = (() => {
    switch (composer.action) {
      case "createTopic":
        return (
          <>
            <Icon name="plus" />
            <span>{t("composer.createTopic")}</span>
          </>
        );
      case "privateMessage":
        return (
          <>
            <Icon name="envelope" />
            <span>{t("composer.createMessage")}</span>
          </>
        );
      case "edit":
        return (
          <>
            <Icon name="pencil" />
            <span>{t("composer.editPost")}</span>
            {composer.topic ? (
              <Link href={topicUrl(composer.topic)} className="topic-link">
                {composer.topic.title}
              </Link>
            ) : null}
          </>
        );
      default:
        return (
          <>
            <Icon name="reply" />
            {composer.topic ? (
              <Link href={topicUrl(composer.topic)} className="topic-link">
                {composer.topic.title}
              </Link>
            ) : null}
            {composer.replyToPost?.username ? (
              <>
                <Icon name="arrow-right" className="text-[var(--primary-low-mid)]" />
                <Link href={userUrl(composer.replyToPost.username)} className="flex items-center gap-1 text-[var(--primary-high)]">
                  {composer.replyToPost.username}
                </Link>
              </>
            ) : null}
          </>
        );
    }
  })();

  const submitLabel =
    composer.action === "createTopic"
      ? t("composer.submitTopic")
      : composer.action === "privateMessage"
        ? t("composer.submitMessage")
        : composer.action === "edit"
          ? t("composer.submitEdit")
          : t("composer.submitReply");
  const submitIcon = composer.action === "createTopic" ? "plus" : composer.action === "privateMessage" ? "envelope" : composer.action === "edit" ? "pencil" : "reply";
  const maxTags = Number(settings.max_tags_per_topic || 5);

  return (
    <div id="reply-control" className="open" style={{ height }}>
      <div className="grippie" onPointerDown={startResize} />
      <div className="reply-area">
        <div className="reply-to">
          <div className="reply-details">{header}</div>
          <div className="composer-controls">
            <button type="button" className="btn btn-flat btn-icon" title={t("composer.minimize")} onClick={() => setMinimized(true)}>
              <Icon name="angles-down" />
            </button>
          </div>
        </div>
        {showTitle || showCategory || composer.action === "privateMessage" ? (
          <div className="composer-fields">
            {composer.action === "privateMessage" ? <UserChooser values={recipients} onChange={setRecipients} placeholder={t("composer.recipients")} /> : null}
            {showTitle ? (
              <div className="title-and-category">
                <input
                  className="d-input title-input"
                  value={title}
                  placeholder={t("composer.titlePlaceholder")}
                  maxLength={Number(settings.max_topic_title_length || 255)}
                  autoFocus={composer.action !== "edit"}
                  onChange={(e) => setTitle(e.target.value)}
                />
                {showCategory ? (
                  <div className="category-input">
                    <CategoryChooser
                      value={categoryId}
                      onChange={setCategoryId}
                      placeholder={t("composer.chooseCategory")}
                      filter={(c) => !c.createRestricted || !!currentUser.admin || !!currentUser.moderator}
                    />
                  </div>
                ) : null}
              </div>
            ) : null}
            {showCategory && settings.tagging_enabled !== "false" ? (
              <div className="tags-row">
                <TagChooser values={tags} onChange={setTags} placeholder={t("composer.tags")} max={maxTags} />
              </div>
            ) : null}
          </div>
        ) : null}
        <div className="wmd-controls">
          <div className="d-editor">
            <div className="d-editor-button-bar" role="toolbar">
              {composer.action === "reply" ? (
                <button type="button" className="btn btn-flat" title={t("topic.quote")} onClick={() => insertAtCaret("> ")}>
                  <Icon name="quote-left" />
                </button>
              ) : null}
              <button type="button" className="btn btn-flat" title={t("composer.bold")} onClick={() => wrap("**", "**", "strong text")}>
                <Icon name="bold" />
              </button>
              <button type="button" className="btn btn-flat" title={t("composer.italic")} onClick={() => wrap("*", "*", "emphasized text")}>
                <Icon name="italic" />
              </button>
              <span className="separator" />
              <button type="button" className="btn btn-flat" title={t("composer.link")} onClick={() => wrap("[", "](https://)", "link text")}>
                <Icon name="link" />
              </button>
              <button type="button" className="btn btn-flat" title={t("composer.blockquote")} onClick={() => prefixLines(() => "> ", "quote")}>
                <Icon name="quote-left" />
              </button>
              <button type="button" className="btn btn-flat" title={t("composer.code")} onClick={() => {
                const el = textareaRef.current;
                const selected = el ? el.value.slice(el.selectionStart, el.selectionEnd) : "";
                if (selected.includes("\n")) {
                  wrap("```\n", "\n```", "");
                } else {
                  wrap("`", "`", "code");
                }
              }}>
                <Icon name="code" />
              </button>
              <button type="button" className="btn btn-flat" title={t("composer.upload")} onClick={() => fileRef.current?.click()}>
                <Icon name="upload" />
              </button>
              <span className="separator" />
              <button type="button" className="btn btn-flat" title={t("composer.bullet")} onClick={() => prefixLines(() => "- ", "list item")}>
                <Icon name="list-ul" />
              </button>
              <button type="button" className="btn btn-flat" title={t("composer.ordered")} onClick={() => prefixLines((i) => `${i + 1}. `, "list item")}>
                <Icon name="list-ol" />
              </button>
              <button type="button" className="btn btn-flat" title={t("composer.heading")} onClick={() => prefixLines(() => "## ", "Heading")}>
                <Icon name="heading" />
              </button>
              <button type="button" className="btn btn-flat" title={t("composer.strikethrough")} onClick={() => wrap("~~", "~~", "strikethrough")}>
                <Icon name="strikethrough" />
              </button>
              {settings.poll_enabled !== "false" ? (
                <button type="button" className="btn btn-flat" title={t("poll.build")} onClick={() => setPollBuilder(true)}>
                  <Icon name="chart-bar" />
                </button>
              ) : null}
              <input
                ref={fileRef}
                type="file"
                multiple
                hidden
                onChange={(e) => {
                  if (e.target.files) {
                    uploadFiles(e.target.files);
                  }
                  e.target.value = "";
                }}
              />
            </div>
            <div className="relative flex flex-1 min-h-0">
              <textarea
                ref={textareaRef}
                className="d-editor-input"
                value={raw}
                placeholder={t("composer.bodyPlaceholder")}
                onChange={(e) => {
                  setRaw(e.target.value);
                  updateMention(e.target.value, e.target.selectionStart);
                }}
                onKeyDown={(e) => {
                  if (mention) {
                    if (e.key === "ArrowDown") {
                      e.preventDefault();
                      setMention({ ...mention, index: (mention.index + 1) % mention.users.length });
                      return;
                    }
                    if (e.key === "ArrowUp") {
                      e.preventDefault();
                      setMention({ ...mention, index: (mention.index - 1 + mention.users.length) % mention.users.length });
                      return;
                    }
                    if (e.key === "Enter" || e.key === "Tab") {
                      e.preventDefault();
                      chooseMention(mention.users[mention.index]);
                      return;
                    }
                    if (e.key === "Escape") {
                      setMention(null);
                      return;
                    }
                  }
                  if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                    e.preventDefault();
                    submit();
                  } else if (e.key === "Escape") {
                    setMinimized(true);
                  }
                }}
                onPaste={(e) => {
                  const files = Array.from(e.clipboardData.files);
                  if (files.length > 0) {
                    e.preventDefault();
                    uploadFiles(files);
                  }
                }}
                onDrop={(e) => {
                  if (e.dataTransfer.files.length > 0) {
                    e.preventDefault();
                    uploadFiles(e.dataTransfer.files);
                  }
                }}
              />
              {mention ? (
                <ul className="mention-autocomplete" style={{ left: 8, bottom: 8 }}>
                  {mention.users.map((user, i) => (
                    <li key={user.id}>
                      <button
                        type="button"
                        className={i === mention.index ? "selected" : ""}
                        onMouseDown={(e) => {
                          e.preventDefault();
                          chooseMention(user);
                        }}
                      >
                        <Avatar user={user} size={20} />
                        <span className="font-bold">{user.username}</span>
                        <span className="text-[var(--primary-medium)]">{user.name}</span>
                      </button>
                    </li>
                  ))}
                </ul>
              ) : null}
            </div>
          </div>
          {showPreview ? (
            <div className="d-editor-preview-wrapper">
              <Cooked html={raw.trim() ? preview : ""} />
            </div>
          ) : null}
        </div>
        <div className="submit-panel">
          <div className="save-or-cancel">
            <button type="button" className="btn btn-primary create" disabled={submitting || uploading > 0} onClick={submit} title="Ctrl+Enter">
              <Icon name={submitIcon} />
              {submitLabel}
            </button>
            <button type="button" className="cancel" onClick={discard}>
              {t("composer.cancel")}
            </button>
            {composer.action === "edit" ? (
              <input className="d-input max-w-[20em]" value={editReason} placeholder={t("composer.editReason")} onChange={(e) => setEditReason(e.target.value)} />
            ) : null}
            {uploading > 0 ? <span className="text-[var(--primary-medium)]">{t("composer.uploading")}</span> : null}
            {error ? <span className="composer-error">{error}</span> : null}
          </div>
          <button type="button" className="btn btn-flat preview-toggle" onClick={() => setShowPreview(!showPreview)}>
            {showPreview ? t("composer.hidePreview") : t("composer.showPreview")}
          </button>
          {composer.author && composer.action === "edit" && composer.author.id !== currentUser.id ? <Avatar user={composer.author} size={24} /> : null}
        </div>
      </div>
      {pollBuilder ? (
        <PollBuilder
          onClose={() => setPollBuilder(false)}
          onInsert={(markdown) => {
            setPollBuilder(false);
            insertAtCaret(markdown);
          }}
        />
      ) : null}
    </div>
  );
}
