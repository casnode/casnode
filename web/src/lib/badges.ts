import type { MessageKey, Translate } from "@/lib/i18n";
import type { Badge } from "@/lib/types";

const badgeTypes: Record<number, string> = { 1: "gold", 2: "silver", 3: "bronze" };

// The system badges are named in the interface language; the others as the admins named them.
export function badgeName(badge: Pick<Badge, "name" | "code" | "system">, t: Translate): string {
  if (badge.system && badge.code) {
    const key = `badge.${badge.code}.name` as MessageKey;
    const name = t(key);
    if (name !== key) {
      return name;
    }
  }
  return badge.name;
}

export function badgeDescription(badge: Pick<Badge, "description" | "code" | "system">, t: Translate): string {
  if (badge.system && badge.code) {
    const key = `badge.${badge.code}.description` as MessageKey;
    const text = t(key);
    if (text !== key) {
      return text;
    }
  }
  return badge.description;
}

export function badgeTypeName(badgeType: number): string {
  return badgeTypes[badgeType] ?? "bronze";
}
