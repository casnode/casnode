"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiGet } from "@/lib/api";
import { formatNumber } from "@/lib/format";
import type { MessageKey } from "@/lib/i18n";
import type { User } from "@/lib/types";
import { userUrl } from "@/lib/utils";

interface Item {
  user: User;
  likesReceived: number;
  likesGiven: number;
  topicCount: number;
  postCount: number;
  topicsEntered: number;
  postsRead: number;
  daysVisited: number;
}

const columns: { order: string; field: keyof Omit<Item, "user">; label: MessageKey; icon?: string }[] = [
  { order: "likes_received", field: "likesReceived", label: "users.received", icon: "heart" },
  { order: "likes_given", field: "likesGiven", label: "users.given", icon: "heart" },
  { order: "topic_count", field: "topicCount", label: "users.topicsCreated" },
  { order: "post_count", field: "postCount", label: "users.repliesPosted" },
  { order: "topics_entered", field: "topicsEntered", label: "users.topicsViewed" },
  { order: "posts_read", field: "postsRead", label: "users.postsRead" },
  { order: "days_visited", field: "daysVisited", label: "users.daysVisited" },
];

// The Users page: everyone with their activity, sortable by any column, like Discourse's directory.
export function Directory({ initial, initialMore }: { initial: Item[]; initialMore: boolean }) {
  const { t } = useSite();
  const [items, setItems] = useState(initial);
  const [more, setMore] = useState(initialMore);
  const [order, setOrder] = useState("likes_received");
  const [asc, setAsc] = useState(false);
  const [name, setName] = useState("");
  const [page, setPage] = useState(0);
  const first = useRef(true);

  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    const timer = setTimeout(async () => {
      const res = await apiGet<Item[], boolean>("get-directory", { order, asc: asc ? "true" : undefined, name, page: 0 });
      setItems(res.data ?? []);
      setMore(!!res.data2);
      setPage(0);
    }, 250);
    return () => clearTimeout(timer);
  }, [order, asc, name]);

  const loadMore = async () => {
    const res = await apiGet<Item[], boolean>("get-directory", { order, asc: asc ? "true" : undefined, name, page: page + 1 });
    setItems((list) => [...list, ...(res.data ?? [])]);
    setMore(!!res.data2);
    setPage(page + 1);
  };

  return (
    <>
      <div className="mb-4 max-w-[20em]">
        <input className="d-input" value={name} placeholder={t("users.filter")} onChange={(e) => setName(e.target.value)} />
      </div>
      <div className="overflow-x-auto">
        <table className="directory-table">
          <thead>
            <tr>
              <th>{t("users.username")}</th>
              {columns.map((col) => (
                <th
                  key={col.order}
                  className={order === col.order ? "active" : ""}
                  onClick={() => {
                    if (order === col.order) {
                      setAsc(!asc);
                    } else {
                      setOrder(col.order);
                      setAsc(false);
                    }
                  }}
                >
                  {col.icon ? <Icon name={col.icon} className="mr-1 text-[var(--love)]" /> : null}
                  {t(col.label)}
                  {order === col.order ? <Icon name={asc ? "chevron-up" : "chevron-down"} className="ml-1" /> : null}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.user.id}>
                <td>
                  <div className="user-info">
                    <Link href={userUrl(item.user.username)}>
                      <Avatar user={item.user} size={45} />
                    </Link>
                    <div>
                      <Link href={userUrl(item.user.username)} className="username block">
                        {item.user.username}
                      </Link>
                      {item.user.name && item.user.name !== item.user.username ? <span className="name">{item.user.name}</span> : null}
                    </div>
                  </div>
                </td>
                {columns.map((col) => (
                  <td key={col.order}>{formatNumber(item[col.field])}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {more ? (
        <button type="button" className="btn btn-default mt-4" onClick={loadMore}>
          {t("common.loadMore")}
        </button>
      ) : null}
    </>
  );
}
