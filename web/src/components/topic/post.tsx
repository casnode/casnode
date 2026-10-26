"use client";

import Link from "next/link";
import { useState } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { useComposer } from "@/components/composer-provider";
import { Cooked } from "@/components/cooked";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { FlagModal, HistoryModal } from "@/components/topic/modals";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { longDate, relativeAge } from "@/lib/format";
import type { MessageKey } from "@/lib/i18n";
import type { Post as PostData, Topic, User, UserMap } from "@/lib/types";
import { topicUrl, userUrl } from "@/lib/utils";

export interface PostProps {
  post: PostData;
  topic: Topic;
  users: UserMap;
  previousPostNumber?: number;
  nextPost?: PostData;
  canReply: boolean;
  highlighted?: boolean;
  onChange: (post: PostData) => void;
  onDeleted: (post: PostData) => void;
  onAddUsers: (users: UserMap) => void;
}

function PostNames({ user }: { user?: User }) {
  if (!user) {
    return null;
  }
  const staff = user.admin || user.moderator;
  return (
    <div className="names trigger-user-card">
      <span className={`first username${staff ? " staff" : ""}${user.moderator ? " moderator" : ""}`}>
        <Link href={userUrl(user.username)} data-user-card={user.username}>
          {user.username}
        </Link>
        {staff ? <Icon name="shield-halved" className="shield" title={user.admin ? "admin" : "moderator"} /> : null}
      </span>
      {user.name && user.name.toLowerCase() !== user.username.toLowerCase() ? (
        <span className="second full-name">
          <Link href={userUrl(user.username)}>{user.name}</Link>
        </span>
      ) : null}
      {user.title ? <span className="user-title">{user.title}</span> : null}
    </div>
  );
}

// Replies to a post, or the post it replies to, shown in place.
function EmbeddedPosts({ posts, users, topic, top }: { posts: PostData[]; users: UserMap; topic: Topic; top?: boolean }) {
  const { locale } = useSite();
  return (
    <section className={`embedded-posts${top ? " top" : ""}`}>
      {posts.map((p) => {
        const author = users[p.userId];
        return (
          <div key={p.id} className="reply" data-post-number={p.postNumber}>
            <Link href={author ? userUrl(author.username) : "#"}>
              <Avatar user={author} size={25} />
            </Link>
            <div className="topic-body">
              <div className="topic-meta-data">
                <PostNames user={author} />
                <div className="post-infos">
                  <Link href={topicUrl(topic, p.postNumber)} className="post-date" suppressHydrationWarning>
                    {relativeAge(p.createdAt, locale)}
                  </Link>
                </div>
              </div>
              <Cooked html={p.cooked} />
            </div>
          </div>
        );
      })}
    </section>
  );
}

export function Post({ post, topic, users, previousPostNumber, nextPost, canReply, highlighted, onChange, onDeleted, onAddUsers }: PostProps) {
  const { t, locale, currentUser, login } = useSite();
  const { open: openComposer } = useComposer();
  const [showMore, setShowMore] = useState(false);
  const [replies, setReplies] = useState<PostData[] | null>(null);
  const [parent, setParent] = useState<PostData[] | null>(null);
  const [likers, setLikers] = useState<User[] | null>(null);
  const [showHidden, setShowHidden] = useState(false);
  const [flagOpen, setFlagOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const author = users[post.userId];
  const replyToUser = users[post.replyToUserId];
  const url = topicUrl(topic, post.postNumber);

  if (post.postType === 3) {
    const actor = users[post.userId];
    return (
      <div className="small-action" id={`post_${post.postNumber}`} data-post-number={post.postNumber}>
        <div className="topic-avatar">
          <Icon name={post.actionCode?.startsWith("closed") ? (post.actionCode.endsWith("enabled") ? "lock" : "circle-check") : post.actionCode?.startsWith("archived") ? "box-archive" : post.actionCode?.startsWith("pinned") ? "thumbtack" : "eye-slash"} />
        </div>
        <div className="small-action-desc">
          {actor ? (
            <Link href={userUrl(actor.username)}>
              <Avatar user={actor} size={25} />
            </Link>
          ) : null}
          <span>
            {t(`action.${post.actionCode}` as MessageKey)}{" "}
            <span title={longDate(post.createdAt, locale)} suppressHydrationWarning>
              {relativeAge(post.createdAt, locale)}
            </span>
          </span>
        </div>
      </div>
    );
  }

  const requireLogin = () => {
    if (!currentUser) {
      login();
      return true;
    }
    return false;
  };

  const toggleLike = async () => {
    if (requireLogin() || busy) {
      return;
    }
    setBusy(true);
    try {
      const res = await apiPost<{ likeCount: number; liked: boolean }>("like-post", { id: post.id, undo: post.liked ? "true" : undefined });
      onChange({ ...post, liked: res.data.liked, likeCount: res.data.likeCount });
      setLikers(null);
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const toggleLikers = async () => {
    if (likers) {
      setLikers(null);
      return;
    }
    const res = await apiGet<User[]>("get-post-likers", { id: post.id });
    setLikers(res.data);
  };

  const toggleBookmark = async () => {
    if (requireLogin()) {
      return;
    }
    try {
      await apiPost("bookmark-post", { id: post.id, undo: post.bookmarked ? "true" : undefined });
      onChange({ ...post, bookmarked: !post.bookmarked });
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const copyLink = async () => {
    const full = `${window.location.origin}${url}`;
    try {
      await navigator.clipboard.writeText(full);
      toast.success(t("topic.linkCopied"));
    } catch {
      window.prompt(t("share.title"), full);
    }
  };

  const edit = async () => {
    const res = await apiGet<PostData>("get-post", { id: post.id });
    openComposer({ action: "edit", post: res.data, topic, author });
  };

  const remove = async () => {
    if (!window.confirm(post.postNumber === 1 ? t("admin.confirmDeleteTopic") : t("admin.confirmDeletePost"))) {
      return;
    }
    try {
      await apiPost("delete-post", { id: post.id });
      onDeleted(post);
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const recover = async () => {
    try {
      await apiPost("recover-post", { id: post.id });
      onChange({ ...post, deleted: false, canRecover: false, canDelete: true });
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const reply = () => {
    if (requireLogin()) {
      return;
    }
    openComposer({ action: "reply", topic, replyToPost: { postNumber: post.postNumber, userId: post.userId, username: author?.username } });
  };

  const toggleReplies = async () => {
    if (replies) {
      setReplies(null);
      return;
    }
    const res = await apiGet<PostData[], UserMap>("get-post-replies", { id: post.id });
    onAddUsers(res.data2);
    setReplies(res.data);
  };

  const toggleParent = async () => {
    if (parent) {
      setParent(null);
      return;
    }
    const res = await apiGet<PostData[]>("get-posts", { topicId: topic.id, postNumbers: String(post.replyToPostNumber) });
    setParent(res.data);
  };

  // a reply right below the post it answers needs no "1 Reply" button, like Discourse
  const repliesDirectlyBelow = post.replyCount === 1 && nextPost?.replyToPostNumber === post.postNumber;
  const showRepliesButton = post.replyCount > 0 && !repliesDirectlyBelow;
  const showReplyTab = post.replyToPostNumber > 0 && post.replyToPostNumber !== previousPostNumber;
  const yours = currentUser?.id === post.userId;
  const hiddenItems = post.canEdit || post.canDelete || post.canRecover || (!!currentUser && !yours);

  return (
    <div
      className={`topic-post${post.deleted ? " deleted" : ""}${highlighted ? " highlighted" : ""}${yours ? " current-user-post" : ""}`}
      id={`post_${post.postNumber}`}
      data-post-number={post.postNumber}
      data-post-id={post.id}
    >
      <article className="boxed onscreen-post" aria-label={`post #${post.postNumber} by @${author?.username ?? ""}`}>
        <div className="row">
          <div className="topic-avatar">
            <Link href={author ? userUrl(author.username) : "#"} className="trigger-user-card" aria-hidden="true" tabIndex={-1}>
              <Avatar user={author} size={45} />
            </Link>
          </div>
          <div className="topic-body clearfix">
            <div className="topic-meta-data" role="heading" aria-level={2}>
              <PostNames user={author} />
              <div className="post-infos">
                {post.version > 1 ? (
                  <div className="post-info edits">
                    <button type="button" title={t("topic.revisions", { count: post.version - 1 })} onClick={() => setHistoryOpen(true)}>
                      {post.version > 2 ? <span>{post.version - 1}</span> : null}
                      <Icon name="pencil" />
                    </button>
                  </div>
                ) : null}
                {showReplyTab ? (
                  <button type="button" className="post-info reply-to-tab" title={t("topic.inReplyTo")} onClick={toggleParent}>
                    <Icon name="share-nodes" className="rotate-180 scale-y-[-1]" />
                    <Avatar user={replyToUser} size={20} />
                    <span>{replyToUser?.username}</span>
                  </button>
                ) : null}
                {post.bookmarked ? (
                  <div className="post-info bookmark">
                    <Icon name="bookmark" className="post-bookmark" />
                  </div>
                ) : null}
                <div className="post-info post-date">
                  <Link href={url} className="post-date" title={longDate(post.createdAt, locale)} data-share-url={url}>
                    <span className="relative-date" suppressHydrationWarning>
                      {relativeAge(post.createdAt, locale)}
                    </span>
                  </Link>
                </div>
              </div>
            </div>
            {parent ? <EmbeddedPosts posts={parent} users={users} topic={topic} top /> : null}
            <div className="regular contents">
              {post.hidden && !showHidden ? (
                <div className="post-hidden-notice">
                  {t("topic.hiddenPost")}{" "}
                  <button type="button" className="btn-link" onClick={() => setShowHidden(true)}>
                    {t("topic.viewHidden")}
                  </button>
                </div>
              ) : (
                <Cooked html={post.cooked} />
              )}
              {likers ? (
                <div className="post-likers">
                  {likers.map((user) => (
                    <Link key={user.id} href={userUrl(user.username)} title={user.username}>
                      <Avatar user={user} size={20} />
                    </Link>
                  ))}
                  <span>{t("topic.likedBy")}</span>
                </div>
              ) : null}
              <section className="post-menu-area clearfix">
                <nav className="post-controls">
                  {showRepliesButton ? (
                    <button type="button" className="show-replies" aria-expanded={!!replies} onClick={toggleReplies}>
                      {post.replyCount === 1 ? t("topic.oneReply") : t("topic.replies", { count: post.replyCount })}
                      <Icon name={replies ? "chevron-up" : "chevron-down"} />
                    </button>
                  ) : null}
                  <div className="actions">
                    {!post.deleted ? (
                      <div className="double-button">
                        {post.likeCount > 0 ? (
                          <button type="button" className="like-count button-count" title={t("topic.likedBy")} onClick={toggleLikers}>
                            {post.likeCount}
                          </button>
                        ) : null}
                        {!yours ? (
                          <button
                            type="button"
                            className={`toggle-like${post.liked ? " has-like" : ""}`}
                            title={post.liked ? t("topic.unlike") : t("topic.like")}
                            aria-pressed={post.liked}
                            onClick={toggleLike}
                          >
                            <Icon name={post.liked ? "heart" : "far-heart"} />
                          </button>
                        ) : post.likeCount > 0 ? (
                          <span className="px-1 text-[var(--primary-low-mid)]">
                            <Icon name="heart" />
                          </span>
                        ) : null}
                      </div>
                    ) : null}
                    <button type="button" className="post-action-menu__copy-link" title={t("topic.copyLink")} onClick={copyLink}>
                      <Icon name="link" />
                    </button>
                    {showMore ? (
                      <>
                        {currentUser && !yours && !post.deleted ? (
                          <button type="button" className={post.flagged ? "has-flagged" : ""} title={t("topic.flagPost")} onClick={() => setFlagOpen(true)} disabled={post.flagged}>
                            <Icon name="flag" />
                          </button>
                        ) : null}
                        {post.canEdit ? (
                          <button type="button" className="edit" title={t("topic.editPost")} onClick={edit}>
                            <Icon name="pencil" />
                          </button>
                        ) : null}
                        {currentUser ? (
                          <button type="button" className={post.bookmarked ? "bookmarked" : ""} title={post.bookmarked ? t("topic.removeBookmark") : t("topic.bookmarkPost")} onClick={toggleBookmark}>
                            <Icon name={post.bookmarked ? "bookmark" : "far-bookmark"} />
                          </button>
                        ) : null}
                        {post.canDelete ? (
                          <button type="button" className="delete" title={t("topic.deletePost")} onClick={remove}>
                            <Icon name="trash-can" />
                          </button>
                        ) : null}
                        {post.canRecover ? (
                          <button type="button" className="recover" title={t("topic.recoverPost")} onClick={recover}>
                            <Icon name="rotate-left" />
                          </button>
                        ) : null}
                      </>
                    ) : hiddenItems || currentUser ? (
                      <button type="button" className="show-more-actions" title={t("topic.moreActions")} onClick={() => setShowMore(true)}>
                        <Icon name="ellipsis" />
                      </button>
                    ) : null}
                    {post.canEdit && !showMore && yours ? (
                      <button type="button" className="edit" title={t("topic.editPost")} onClick={edit}>
                        <Icon name="pencil" />
                      </button>
                    ) : null}
                    {canReply && !post.deleted ? (
                      <button type="button" className="reply create fade-out" title={t("topic.replyToPost")} onClick={reply}>
                        <Icon name="reply" />
                        <span className="d-button-label">{t("topic.reply")}</span>
                      </button>
                    ) : null}
                  </div>
                </nav>
              </section>
              {replies ? <EmbeddedPosts posts={replies} users={users} topic={topic} /> : null}
            </div>
          </div>
        </div>
      </article>
      {flagOpen ? (
        <FlagModal
          post={post}
          onClose={() => setFlagOpen(false)}
          onFlagged={() => {
            setFlagOpen(false);
            onChange({ ...post, flagged: true });
          }}
        />
      ) : null}
      {historyOpen ? <HistoryModal post={post} onClose={() => setHistoryOpen(false)} /> : null}
    </div>
  );
}
