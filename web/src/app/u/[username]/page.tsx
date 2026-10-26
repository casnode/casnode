import type { Metadata } from "next";
import { SummaryPage } from "./summary/summary";
import { loadUser } from "./user-data";

type Props = { params: Promise<{ username: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const user = await loadUser(decodeURIComponent((await params).username));
  if (!user) {
    return {};
  }
  return {
    title: user.name && user.name !== user.username ? `${user.username} (${user.name})` : user.username,
    description: user.bioCooked?.replace(/<[^>]+>/g, " ").trim().slice(0, 200) || undefined,
    alternates: { canonical: `/u/${encodeURIComponent(user.username)}/summary` },
    openGraph: { type: "profile", images: user.avatar ? [user.avatar] : undefined },
  };
}

export default async function Page({ params }: Props) {
  return <SummaryPage username={decodeURIComponent((await params).username)} />;
}
