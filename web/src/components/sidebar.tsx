"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState, type ReactNode } from "react";
import { Icon } from "@/components/icon";
import { ChatChannelLinks } from "@/components/chat/chat";
import { useChat } from "@/components/chat/chat-provider";
import { useLayoutState } from "@/components/layout-state";
import { useSite } from "@/components/site-provider";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { locales } from "@/lib/i18n";
import { categoryUrl, isStaff, tagUrl, userUrl } from "@/lib/utils";

function SectionLink({ href, active, prefix, children, badge, secondary }: { href: string; active: boolean; prefix: ReactNode; children: ReactNode; badge?: ReactNode; secondary?: boolean }) {
  const { hideSidebarOnMobile } = useLayoutState();
  return (
    <li className="sidebar-section-link-wrapper">
      <Link href={href} className={`sidebar-section-link${active ? " active" : ""}${secondary ? " secondary" : ""}`} onClick={hideSidebarOnMobile}>
        <span className="sidebar-section-link-prefix">{prefix}</span>
        <span className="sidebar-section-link-content-text">{children}</span>
        {badge}
      </Link>
    </li>
  );
}

function Section({ id, title, children, action }: { id: string; title?: string; children: ReactNode; action?: ReactNode }) {
  const [collapsed, setCollapsed] = useState(false);
  return (
    <div className={`sidebar-section-wrapper${collapsed ? " collapsed" : ""}`} data-section-name={id}>
      {title ? (
        <div className="sidebar-section-header-wrapper">
          <button type="button" className="sidebar-section-header" aria-expanded={!collapsed} onClick={() => setCollapsed(!collapsed)}>
            <span className="sidebar-section-header-caret">
              <Icon name="angle-down" />
            </span>
            <span className="sidebar-section-header-text">{title}</span>
          </button>
          {action}
        </div>
      ) : null}
      <ul className="sidebar-section-content">{children}</ul>
    </div>
  );
}

export function Sidebar() {
  const { currentUser, categories, site, t, locale, settings } = useSite();
  const { sidebarVisible, mobileSidebarOpen, hideSidebarOnMobile } = useLayoutState();
  const chat = useChat();
  const pathname = usePathname();
  const router = useRouter();

  const counts = site.counts ?? {};
  const topics = ["/", "/latest", "/new", "/unread", "/top", "/hot"].includes(pathname);
  // like Discourse, Uncategorized only shows up once it has topics
  const uncategorizedId = Number(settings.uncategorized_category_id || 0);
  const topCategories = categories.filter((c) => c.parentId === 0 && !(c.id === uncategorizedId && c.topicCount === 0)).slice(0, 12);
  const tags = (site.topTags ?? []).slice(0, 8);
  const staff = isStaff(currentUser);

  const setLocale = (value: string) => {
    document.cookie = `locale=${value}; path=/; max-age=31536000; samesite=lax`;
    router.refresh();
  };

  return (
    <div className={`sidebar-wrapper${sidebarVisible ? "" : " desktop-hidden"}${mobileSidebarOpen ? " mobile-open" : ""}`}>
      <aside className="sidebar-container" aria-label="sidebar">
        <div className="sidebar-sections">
          <Section id="community">
            <SectionLink href="/latest" active={topics} prefix={<Icon name="layer-group" />}>
              {t("sidebar.topics")}
            </SectionLink>
            {currentUser ? (
              <SectionLink href={userUrl(currentUser.username, "activity")} active={pathname === userUrl(currentUser.username, "activity")} prefix={<Icon name="user" />}>
                {t("sidebar.myPosts")}
              </SectionLink>
            ) : null}
            {staff ? (
              <SectionLink
                href="/review"
                active={pathname.startsWith("/review")}
                prefix={<Icon name="flag" />}
                badge={counts.reviewables ? <span className="sidebar-section-link-content-badge has-new">{counts.reviewables}</span> : null}
              >
                {t("sidebar.review")}
              </SectionLink>
            ) : null}
            {currentUser?.admin ? (
              <SectionLink href="/admin" active={pathname.startsWith("/admin")} prefix={<Icon name="wrench" />}>
                {t("sidebar.admin")}
              </SectionLink>
            ) : null}
            <li className="sidebar-section-link-wrapper">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button type="button" className="sidebar-section-link">
                    <span className="sidebar-section-link-prefix">
                      <Icon name="ellipsis" className="rotate-90" />
                    </span>
                    <span className="sidebar-section-link-content-text">{t("sidebar.more")}</span>
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="min-w-[14em]">
                  <DropdownMenuItem asChild>
                    <Link href="/about">
                      <Icon name="circle-info" />
                      {t("sidebar.about")}
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href="/faq">
                      <Icon name="circle-question" />
                      {t("sidebar.faq")}
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href="/u">
                      <Icon name="users" />
                      {t("sidebar.users")}
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href="/g">
                      <Icon name="user-group" />
                      {t("sidebar.groups")}
                    </Link>
                  </DropdownMenuItem>
                  {settings.enable_badges !== "false" ? (
                    <DropdownMenuItem asChild>
                      <Link href="/badges">
                        <Icon name="certificate" />
                        {t("sidebar.badges")}
                      </Link>
                    </DropdownMenuItem>
                  ) : null}
                  <DropdownMenuItem
                    onSelect={() => document.dispatchEvent(new KeyboardEvent("keydown", { key: "?", bubbles: true }))}
                  >
                    <Icon name="keyboard" />
                    {t("shortcuts.title")}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </li>
          </Section>

          {/* the chat pages list the channels themselves */}
          {chat.enabled && !pathname.startsWith("/chat") ? <ChatChannelLinks onNavigate={hideSidebarOnMobile} /> : null}

          <Section
            id="categories"
            title={t("sidebar.categories")}
            action={
              currentUser?.admin ? (
                <Link href="/new-category" className="sidebar-section-header-button" title={t("categories.newCategory")}>
                  <Icon name="plus" />
                </Link>
              ) : null
            }
          >
            {topCategories.map((category) => {
              const href = categoryUrl(category, categories);
              return (
                <SectionLink
                  key={category.id}
                  href={href}
                  active={pathname.startsWith(href)}
                  prefix={
                    category.readRestricted ? (
                      <Icon name="lock" />
                    ) : (
                      <span className="prefix-square" style={{ background: `#${category.color}` }} />
                    )
                  }
                >
                  {category.name}
                </SectionLink>
              );
            })}
            <SectionLink href="/categories" active={pathname === "/categories"} prefix={<Icon name="list" />} secondary>
              {t("sidebar.allCategories")}
            </SectionLink>
          </Section>

          {settings.tagging_enabled !== "false" ? (
            <Section id="tags" title={t("sidebar.tags")}>
              {tags.map((tag) => (
                <SectionLink key={tag.id} href={tagUrl(tag.name)} active={pathname === tagUrl(tag.name)} prefix={<Icon name="tag" />}>
                  {tag.name}
                </SectionLink>
              ))}
              <SectionLink href="/tags" active={pathname === "/tags"} prefix={<Icon name="list" />} secondary>
                {t("sidebar.allTags")}
              </SectionLink>
            </Section>
          ) : null}

          {currentUser && settings.enable_personal_messages !== "false" ? (
            <Section id="messages" title={t("sidebar.messages")}>
              <SectionLink
                href={userUrl(currentUser.username, "messages")}
                active={pathname.startsWith(userUrl(currentUser.username, "messages"))}
                prefix={<Icon name="inbox" />}
                badge={counts.unreadMessages ? <span className="sidebar-section-link-content-badge has-new">{counts.unreadMessages}</span> : null}
              >
                {t("sidebar.inbox")}
              </SectionLink>
            </Section>
          ) : null}
        </div>
        <div className="sidebar-footer-wrapper">
          <div className="sidebar-footer-container">
            <Icon name="language" />
            <select className="bg-transparent text-[var(--primary-medium)] outline-none" value={locale} onChange={(e) => setLocale(e.target.value)} aria-label={t("sidebar.language")}>
              {locales.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </select>
            <a href="https://github.com/casnode/casnode" target="_blank" rel="noopener" className="ml-auto">
              {t("common.poweredBy")}
            </a>
          </div>
        </div>
      </aside>
    </div>
  );
}
