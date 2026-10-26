"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { apiGet } from "@/lib/api";
import type { SearchResult } from "@/lib/types";
import { categoryUrl, tagUrl, topicUrl, userUrl } from "@/lib/utils";

// The search panel under the header: results appear while typing, Enter opens the full search page.
export function SearchMenu({ onClose }: { onClose: () => void }) {
  const { t, categories } = useSite();
  const router = useRouter();
  const [term, setTerm] = useState("");
  const [found, setResult] = useState<SearchResult | null>(null);
  const [loading, setLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  useEffect(() => {
    const q = term.trim();
    if (q.length < 2) {
      return;
    }
    const timer = setTimeout(async () => {
      setLoading(true);
      try {
        const res = await apiGet<SearchResult>("search", { q, limit: 6 });
        setResult(res.data);
      } catch {
        setResult(null);
      } finally {
        setLoading(false);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [term]);

  const submit = () => {
    if (term.trim()) {
      onClose();
      router.push(`/search?q=${encodeURIComponent(term.trim())}`);
    }
  };

  const result = term.trim().length >= 2 ? found : null;
  const hasResults = result && (result.posts.length > 0 || result.matchUsers.length > 0 || result.categories.length > 0 || result.tags.length > 0);

  return (
    <div className="search-menu menu-panel">
      <div className="search-input">
        <Icon name="search" />
        <input
          ref={inputRef}
          type="text"
          value={term}
          placeholder={t("search.placeholder")}
          aria-label={t("search.placeholder")}
          onChange={(e) => setTerm(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              submit();
            }
          }}
        />
        {loading ? <span className="spinner small" /> : null}
        {term ? (
          <button type="button" className="btn btn-flat btn-icon" onClick={() => setTerm("")} aria-label="clear">
            <Icon name="xmark" />
          </button>
        ) : null}
        <Link href={`/search${term ? `?q=${encodeURIComponent(term)}` : ""}`} className="btn btn-flat btn-icon" onClick={onClose} title={t("search.advanced")}>
          <Icon name="gear" />
        </Link>
      </div>
      <div className="results">
        {!result ? <div className="search-tips">{t("search.tips")}</div> : null}
        {result && !hasResults && !loading ? <div className="search-tips">{t("search.noResults")}</div> : null}
        {result && result.posts.length > 0 ? (
          <div className="search-result-group">
            <div className="heading">{t("search.topicsPosts")}</div>
            {result.posts.map((post) => {
              const topic = result.topics[post.topicId];
              if (!topic) {
                return null;
              }
              return (
                <Link key={post.id} href={topicUrl(topic, post.postNumber)} className="search-link" onClick={onClose}>
                  <span className="topic-title">{topic.title}</span>
                  <div className="mt-0.5">
                    <CategoryBadge category={categories.find((c) => c.id === topic.categoryId)} categories={categories} link={false} />
                  </div>
                  <span className="blurb">{post.blurb}</span>
                </Link>
              );
            })}
            {result.more ? (
              <button type="button" className="search-link w-full text-left" onClick={submit}>
                {t("search.moreResults")}
              </button>
            ) : null}
          </div>
        ) : null}
        {result && result.matchUsers.length > 0 ? (
          <div className="search-result-group">
            <div className="heading">{t("search.users")}</div>
            {result.matchUsers.map((user) => (
              <Link key={user.id} href={userUrl(user.username)} className="search-link flex items-center gap-2" onClick={onClose}>
                <Avatar user={user} size={24} />
                <span className="font-bold">{user.username}</span>
                {user.name && user.name !== user.username ? <span className="text-[var(--primary-medium)]">{user.name}</span> : null}
              </Link>
            ))}
          </div>
        ) : null}
        {result && (result.categories.length > 0 || result.tags.length > 0) ? (
          <div className="search-result-group">
            <div className="heading">{t("search.categoriesTags")}</div>
            {result.categories.map((category) => (
              <Link key={category.id} href={categoryUrl(category, categories)} className="search-link" onClick={onClose}>
                <CategoryBadge category={category} categories={categories} link={false} />
              </Link>
            ))}
            {result.tags.map((tag) => (
              <Link key={tag.id} href={tagUrl(tag.name)} className="search-link flex items-center gap-2" onClick={onClose}>
                <Icon name="tag" className="text-[var(--primary-medium)]" />
                {tag.name}
              </Link>
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}
