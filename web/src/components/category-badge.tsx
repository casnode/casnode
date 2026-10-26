import Link from "next/link";
import type { Category } from "@/lib/types";
import { categoryUrl } from "@/lib/utils";
import { Icon } from "@/components/icon";

// The small colored square and name that stands for a category in lists and titles; a subcategory shows its parent first.
export function CategoryBadge({ category, categories, link = true, hideParent = false }: { category?: Category | null; categories: Category[]; link?: boolean; hideParent?: boolean }) {
  if (!category) {
    return null;
  }
  const parent = !hideParent && category.parentId ? categories.find((c) => c.id === category.parentId) : undefined;
  const badge = (c: Category) => (
    <span className="badge-category" style={{ ["--category-badge-color" as string]: `#${c.color}` }} title={plainDescription(c.description)}>
      {c.readRestricted ? <Icon name="lock" className="lock" /> : null}
      <span className="badge-category__name">{c.name}</span>
    </span>
  );
  const wrap = (c: Category) =>
    link ? (
      <Link key={c.id} href={categoryUrl(c, categories)} className="badge-category__wrapper">
        {badge(c)}
      </Link>
    ) : (
      <span key={c.id} className="badge-category__wrapper">
        {badge(c)}
      </span>
    );

  if (parent) {
    return (
      <span className="inline-flex items-center gap-2 min-w-0">
        {wrap(parent)}
        {wrap(category)}
      </span>
    );
  }
  return wrap(category);
}

export function plainDescription(description?: string): string {
  return (description ?? "").replace(/<[^>]+>/g, "").trim();
}
