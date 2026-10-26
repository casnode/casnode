"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { CategoryBadge } from "@/components/category-badge";
import { Icon } from "@/components/icon";
import { useLayoutState } from "@/components/layout-state";
import { SearchMenu } from "@/components/search-menu";
import { useSite } from "@/components/site-provider";
import { UserMenu } from "@/components/user-menu";

export function SiteHeader() {
  const { currentUser, settings, site, t, login, categoryById, categories } = useSite();
  const { toggleSidebar, sidebarVisible, headerTopic } = useLayoutState();
  const [panel, setPanel] = useState<"" | "search" | "user">("");
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!panel) {
      return;
    }
    const onDown = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setPanel("");
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setPanel("");
      }
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [panel]);

  const title = settings.title || "Casnode";
  const logo = settings.logo_url;
  const counts = site.counts ?? {};
  const unread = counts.notifications ?? 0;
  const highPriority = counts.highPriorityNotifications ?? 0;
  const topicCategory = headerTopic ? categoryById(headerTopic.categoryId) : undefined;

  return (
    <div className="d-header-wrap">
      <header className="d-header">
        <div className="wrap">
          <div className="contents">
            <span className="header-sidebar-toggle">
              <button
                type="button"
                className={`header-icon-button${sidebarVisible ? "" : ""}`}
                aria-label={t("header.menu")}
                aria-expanded={sidebarVisible}
                onClick={toggleSidebar}
              >
                <Icon name="bars" />
              </button>
            </span>
            <div className="title">
              <Link href="/" aria-label={title}>
                {logo ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={logo} alt={title} className="logo-big" />
                ) : (
                  <span className="site-title-text">{title}</span>
                )}
              </Link>
            </div>
            {headerTopic ? (
              <div className="header-topic-title">
                <Link href={headerTopic.url} className="topic-link">
                  {headerTopic.title}
                </Link>
                {topicCategory ? (
                  <div>
                    <CategoryBadge category={topicCategory} categories={categories} />
                  </div>
                ) : null}
              </div>
            ) : null}
            <div className="panel" ref={panelRef}>
              {!currentUser ? (
                <span className="header-buttons">
                  <button type="button" className="btn btn-primary btn-small sign-up-button" onClick={() => login(true)}>
                    {t("header.signup")}
                  </button>
                  <button type="button" className="btn btn-primary btn-small login-button" onClick={() => login(false)}>
                    <Icon name="user" />
                    {t("header.login")}
                  </button>
                </span>
              ) : null}
              <ul className="d-header-icons" role="navigation">
                <li>
                  <button
                    type="button"
                    className={`header-icon-button search-dropdown${panel === "search" ? " active" : ""}`}
                    aria-label={t("header.search")}
                    title={t("header.search")}
                    onClick={() => setPanel(panel === "search" ? "" : "search")}
                  >
                    <Icon name="search" />
                  </button>
                </li>
                {currentUser ? (
                  <li>
                    <button
                      type="button"
                      className={`header-icon-button current-user${panel === "user" ? " active" : ""}`}
                      aria-label={t("header.notifications")}
                      onClick={() => setPanel(panel === "user" ? "" : "user")}
                    >
                      <Avatar user={currentUser} size={32} />
                      {highPriority > 0 ? (
                        <span className="badge-notification unread-high-priority-notifications">{highPriority}</span>
                      ) : unread > 0 ? (
                        <span className="badge-notification unread-notifications">{unread}</span>
                      ) : null}
                    </button>
                  </li>
                ) : null}
              </ul>
              {panel === "search" ? <SearchMenu onClose={() => setPanel("")} /> : null}
              {panel === "user" && currentUser ? <UserMenu onClose={() => setPanel("")} /> : null}
            </div>
          </div>
        </div>
      </header>
    </div>
  );
}
