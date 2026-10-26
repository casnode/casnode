"use client";

import Link from "next/link";
import { Icon } from "@/components/icon";
import { badgeDescription, badgeName, badgeTypeName } from "@/lib/badges";
import { useSite } from "@/components/site-provider";
import type { Badge } from "@/lib/types";
import { badgeUrl } from "@/lib/utils";

export function BadgeIcon({ badge }: { badge: Pick<Badge, "icon" | "imageUrl"> }) {
  if (badge.imageUrl) {
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={badge.imageUrl} alt="" className="badge-image" />;
  }
  return <Icon name={badge.icon || "certificate"} />;
}

// A badge as a small labelled button, like Discourse's user-badge.
export function BadgeChip({ badge, count, href }: { badge: Badge; count?: number; href?: string }) {
  const { t } = useSite();
  return (
    <Link href={href ?? badgeUrl(badge)} className={`user-badge badge-type-${badgeTypeName(badge.badgeType)}`} title={badgeDescription(badge, t)}>
      <BadgeIcon badge={badge} />
      <span className="badge-display-name">{badgeName(badge, t)}</span>
      {count && count > 1 ? <span className="count">× {count}</span> : null}
    </Link>
  );
}
