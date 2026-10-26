"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Avatar } from "@/components/avatar";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import { longDate } from "@/lib/format";
import type { Poll as PollData, User } from "@/lib/types";
import { isValidDate } from "@/lib/utils";

// A poll of a post, like Discourse's: a single choice votes at once, a multiple choice with the Vote button; the
// results show after voting, or when the poll's setting allows them.
export function Poll({ poll, postId, onChange }: { poll: PollData; postId: number; onChange: (poll: PollData) => void }) {
  const { t, locale, currentUser, login } = useSite();
  const [chosen, setChosen] = useState<string[]>(poll.userVotes ?? []);
  const [prevVotes, setPrevVotes] = useState(poll.userVotes);
  const [showResults, setShowResults] = useState(false);
  const [voters, setVoters] = useState<Record<string, User[]>>({});
  const [busy, setBusy] = useState(false);

  if (prevVotes !== poll.userVotes) {
    setPrevVotes(poll.userVotes);
    setChosen(poll.userVotes ?? []);
  }

  const voted = (poll.userVotes ?? []).length > 0;
  const closed = poll.status === "closed";
  const results = !poll.resultsHidden && (closed || voted || showResults);
  const multiple = poll.type === "multiple";
  const max = poll.max > 0 ? Math.min(poll.max, poll.options.length) : poll.options.length;
  const min = poll.min > 0 ? poll.min : 1;
  const totalVotes = poll.options.reduce((sum, o) => sum + o.votes, 0);

  useEffect(() => {
    if (!results || !poll.public || !currentUser) {
      return;
    }
    let cancelled = false;
    Promise.all(
      poll.options.filter((o) => o.votes > 0).map(async (o) => [o.id, (await apiGet<User[]>("get-poll-voters", { postId, name: poll.name, option: o.id })).data] as const),
    )
      .then((pairs) => {
        if (!cancelled) {
          setVoters(Object.fromEntries(pairs));
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [results, poll, postId, currentUser]);

  const call = async (fn: () => Promise<{ data: PollData }>) => {
    if (!currentUser) {
      login();
      return;
    }
    setBusy(true);
    try {
      const res = await fn();
      onChange(res.data);
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const vote = (options: string[]) => call(() => apiPost<PollData>("vote-poll", undefined, { postId, name: poll.name, options }));

  const choose = (id: string) => {
    if (closed || busy) {
      return;
    }
    if (!multiple) {
      vote([id]);
      return;
    }
    setChosen((list) => (list.includes(id) ? list.filter((x) => x !== id) : list.length < max ? [...list, id] : list));
  };

  return (
    <div className={`poll-container${closed ? " closed" : ""}`}>
      <div className="poll-body">
        {results ? (
          <ul className="results">
            {poll.options.map((option) => {
              const percent = totalVotes > 0 ? Math.round((option.votes * 100) / (multiple ? Math.max(poll.voters, 1) : totalVotes)) : 0;
              const mine = (poll.userVotes ?? []).includes(option.id);
              return (
                <li key={option.id} className={mine ? "chosen" : ""}>
                  <div className="option">
                    <p>
                      <span className="percentage">{percent}%</span>
                      <span dangerouslySetInnerHTML={{ __html: option.html }} />
                      {mine ? <Icon name="check" className="ml-1 text-[var(--success)]" /> : null}
                    </p>
                  </div>
                  <div className="bar-back">
                    <div className="bar" style={{ width: `${percent}%` }} />
                  </div>
                  {poll.public && voters[option.id]?.length ? (
                    <div className="poll-voters">
                      {voters[option.id].map((user) => (
                        <a key={user.id} href={`/u/${encodeURIComponent(user.username)}`} data-user-card={user.username} title={user.username}>
                          <Avatar user={user} size={20} />
                        </a>
                      ))}
                    </div>
                  ) : null}
                </li>
              );
            })}
          </ul>
        ) : (
          <ul className="options">
            {poll.options.map((option) => {
              const on = chosen.includes(option.id);
              return (
                <li key={option.id}>
                  <button type="button" className={`poll-option${on ? " chosen" : ""}`} disabled={closed || busy} aria-pressed={on} onClick={() => choose(option.id)}>
                    <Icon name={multiple ? (on ? "square-check" : "far-square") : on ? "circle-dot" : "far-circle"} />
                    <span dangerouslySetInnerHTML={{ __html: option.html }} />
                  </button>
                </li>
              );
            })}
          </ul>
        )}
        <div className="poll-info">
          <div className="info-number">{poll.voters}</div>
          <div className="info-label">{t("poll.voters")}</div>
          {multiple && !closed && !results ? <div className="info-text">{min === max ? t("poll.chooseExactly", { count: max }) : t("poll.chooseBetween", { min, max })}</div> : null}
          {poll.resultsHidden && poll.results === "on_vote" && !closed ? <div className="info-text">{t("poll.resultsOnVote")}</div> : null}
          {poll.resultsHidden && poll.results === "on_close" ? <div className="info-text">{t("poll.resultsOnClose")}</div> : null}
          {poll.resultsHidden && poll.results === "staff_only" ? <div className="info-text">{t("poll.resultsStaffOnly")}</div> : null}
          {poll.public ? <div className="info-text">{t("poll.public")}</div> : null}
          {closed ? (
            <div className="info-text">{t("poll.closed")}</div>
          ) : isValidDate(poll.closeAt) ? (
            <div className="info-text" suppressHydrationWarning>
              {t("poll.closesAt", { date: longDate(poll.closeAt, locale) })}
            </div>
          ) : null}
        </div>
      </div>
      <div className="poll-buttons">
        {multiple && !closed && !results ? (
          <button type="button" className="btn btn-primary" disabled={busy || chosen.length < min || chosen.length > max} onClick={() => vote(chosen)}>
            <Icon name="check" />
            {t("poll.vote")}
          </button>
        ) : null}
        {!closed && !voted && !poll.resultsHidden ? (
          <button type="button" className="btn btn-default" onClick={() => setShowResults(!showResults)}>
            <Icon name={showResults ? "eye-slash" : "chart-bar"} />
            {showResults ? t("poll.hideResults") : t("poll.showResults")}
          </button>
        ) : null}
        {!closed && voted ? (
          <button type="button" className="btn btn-flat" disabled={busy} onClick={() => call(() => apiPost<PollData>("remove-poll-vote", { postId, name: poll.name }))}>
            <Icon name="xmark" />
            {t("poll.removeVote")}
          </button>
        ) : null}
        {poll.canClose ? (
          <button
            type="button"
            className="btn btn-flat ml-auto"
            disabled={busy}
            onClick={() => call(() => apiPost<PollData>("set-poll-status", { postId, name: poll.name, status: closed ? "open" : "closed" }))}
          >
            <Icon name={closed ? "unlock" : "lock"} />
            {closed ? t("poll.open") : t("poll.close")}
          </button>
        ) : null}
      </div>
    </div>
  );
}
