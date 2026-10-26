"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { CategoryBadge, plainDescription } from "@/components/category-badge";
import { useComposer } from "@/components/composer-provider";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { apiPost } from "@/lib/api";
import type { MessageKey } from "@/lib/i18n";
import type { Category } from "@/lib/types";
import { categoryUrl, tagUrl } from "@/lib/utils";

const filters = ["latest", "new", "unread", "hot", "top", "categories"] as const;
const periods = ["all", "yearly", "quarterly", "monthly", "weekly", "daily"] as const;

// listUrl is where a navigation pill goes in the current category or tag.
export function listUrl(filter: string, category: Category | undefined, categories: Category[], tag?: string): string {
  if (filter === "categories") {
    return "/categories";
  }
  if (category) {
    return `${categoryUrl(category, categories)}/l/${filter}`;
  }
  if (tag) {
    return `${tagUrl(tag)}/l/${filter}`;
  }
  return `/${filter}`;
}

function CategoryDrop({ category, parent }: { category?: Category; parent?: Category }) {
  const { categories, t } = useSite();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [term, setTerm] = useState("");
  const options = categories.filter((c) => (parent ? c.parentId === parent.id : c.parentId === 0)).filter((c) => !term || c.name.toLowerCase().includes(term.toLowerCase()));
  const label = parent ? t("nav.subcategories") : t("nav.allCategories");

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button type="button" className="select-kit-header category-drop-header">
          <span className="selected-name">{category ? <CategoryBadge category={category} categories={categories} link={false} hideParent /> : label}</span>
          <Icon name={open ? "caret-down" : "chevron-right"} />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-[22em] p-0">
        <div className="dropdown-filter">
          <input autoFocus value={term} placeholder={label} onChange={(e) => setTerm(e.target.value)} />
        </div>
        <div className="max-h-[22em] overflow-y-auto py-1">
          {category ? (
            <button
              type="button"
              className="dropdown-menu-item"
              onClick={() => {
                setOpen(false);
                router.push(parent ? categoryUrl(parent, categories) : "/latest");
              }}
            >
              <span className="item-text">
                <span>{parent ? parent.name : t("nav.allCategories")}</span>
              </span>
            </button>
          ) : null}
          {options.map((c) => (
            <button
              key={c.id}
              type="button"
              className={`dropdown-menu-item${c.id === category?.id ? " is-selected" : ""}`}
              onClick={() => {
                setOpen(false);
                router.push(categoryUrl(c, categories));
              }}
            >
              <span className="item-text">
                <span className="flex items-center gap-2">
                  <CategoryBadge category={c} categories={categories} link={false} hideParent />
                  <span className="text-[var(--primary-medium)] text-[0.8706em]">× {c.topicCount}</span>
                </span>
                {c.description ? <span className="category-row-description">{plainDescription(c.description)}</span> : null}
              </span>
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

function TagDrop({ tag, category }: { tag?: string; category?: Category }) {
  const { site, t, categories } = useSite();
  const router = useRouter();
  const tags = site.topTags ?? [];
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="select-kit-header tag-drop-header">
          <span className="selected-name">{tag ?? t("nav.allTags")}</span>
          <Icon name="chevron-right" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="max-h-[22em]">
        {tag ? (
          <DropdownMenuItem onSelect={() => router.push(category ? categoryUrl(category, categories) : "/latest")}>{t("nav.allTags")}</DropdownMenuItem>
        ) : null}
        {tags.map((item) => (
          <DropdownMenuItem key={item.id} onSelect={() => router.push(tagUrl(item.name))}>
            <span>{item.name}</span>
            <span className="ml-auto text-[var(--primary-medium)] text-[0.8706em]">× {item.topicCount}</span>
          </DropdownMenuItem>
        ))}
        <DropdownMenuItem onSelect={() => router.push("/tags")}>
          <Icon name="list" />
          {t("sidebar.allTags")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function NavigationBar({ filter, category, tag, period }: { filter: string; category?: Category; tag?: string; period?: string }) {
  const { settings, currentUser, categories, site, t } = useSite();
  const { open } = useComposer();
  const router = useRouter();
  const counts = site.counts ?? {};
  const topMenu = (settings.top_menu || "latest|new|unread|hot|top|categories").split("|").filter((f) => (filters as readonly string[]).includes(f));
  const menu = topMenu.filter((f) => currentUser || (f !== "new" && f !== "unread")).filter((f) => !(f === "categories" && (category || tag)));
  const parent = category?.parentId ? categories.find((c) => c.id === category.parentId) : undefined;
  const canCreate = !category || !category.createRestricted || currentUser?.admin || currentUser?.moderator;

  const dismiss = async () => {
    await apiPost("dismiss-topics", { filter, categoryId: category?.id ?? 0 });
    router.refresh();
  };

  return (
    <div className="list-controls">
      <div className="navigation-container">
        <ol className="category-breadcrumb">
          {parent ? (
            <li>
              <CategoryDrop category={parent} />
            </li>
          ) : null}
          <li>
            <CategoryDrop category={category} parent={parent} />
          </li>
          {category && !parent && (category.subcategoryIds?.length ?? 0) > 0 ? (
            <li>
              <CategoryDrop parent={category} />
            </li>
          ) : null}
          {settings.tagging_enabled !== "false" ? (
            <li>
              <TagDrop tag={tag} category={category} />
            </li>
          ) : null}
        </ol>
        <ul className="nav-pills" role="navigation">
          {menu.map((f) => {
            const count = f === "new" ? counts.new : f === "unread" ? counts.unread : 0;
            return (
              <li key={f}>
                <Link href={listUrl(f, category, categories, tag)} className={filter === f ? "active" : ""} title={t(`nav.${f}.title` as MessageKey)}>
                  {t(`nav.${f}` as MessageKey)}
                  {count ? ` (${count})` : ""}
                </Link>
              </li>
            );
          })}
        </ul>
        <div className="navigation-controls">
          {(filter === "new" || filter === "unread") && currentUser ? (
            <button type="button" className="btn btn-default dismiss-read" onClick={dismiss}>
              <Icon name="check" />
              {t("nav.dismiss")}
            </button>
          ) : null}
          {currentUser && canCreate ? (
            <button
              type="button"
              id="create-topic"
              className="btn btn-default"
              onClick={() => open({ action: "createTopic", categoryId: category?.id })}
            >
              <Icon name="plus" />
              {t("nav.newTopic")}
            </button>
          ) : null}
          {currentUser?.admin && category ? (
            <Link href={`${categoryUrl(category, categories)}/edit`} className="btn btn-default btn-icon" title={t("categories.editCategory")}>
              <Icon name="wrench" />
            </Link>
          ) : null}
        </div>
      </div>
      {filter === "top" ? (
        <div className="mb-3 flex flex-wrap items-center gap-2">
          {periods.map((p) => {
            const key = `list.period${p.charAt(0).toUpperCase()}${p.slice(1)}` as MessageKey;
            const base = listUrl("top", category, categories, tag);
            return (
              <Link key={p} href={p === "all" ? base : `${base}?period=${p}`} className={`btn btn-small ${period === p || (!period && p === "all") ? "btn-primary" : "btn-default"}`}>
                {t(key)}
              </Link>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
