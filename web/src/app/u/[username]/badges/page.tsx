import { UserBadges } from "@/components/badge/badges";
import { serverGet } from "@/lib/server-api";
import type { UserBadgeSummary } from "@/lib/types";

export default async function Page({ params }: { params: Promise<{ username: string }> }) {
  const username = decodeURIComponent((await params).username);
  const res = await serverGet<UserBadgeSummary[]>("get-user-badges", { username });
  return <UserBadges badges={res.data ?? []} />;
}
