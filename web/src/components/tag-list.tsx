import Link from "next/link";
import { tagUrl } from "@/lib/utils";

export function TagList({ tags }: { tags?: string[] | null }) {
  if (!tags || tags.length === 0) {
    return null;
  }
  return (
    <div className="discourse-tags" role="list" aria-label="tags">
      {tags.map((tag) => (
        <Link key={tag} href={tagUrl(tag)} className="discourse-tag" role="listitem" data-tag-name={tag}>
          {tag}
        </Link>
      ))}
    </div>
  );
}
