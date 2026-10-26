import type { Metadata } from "next";
import { ActivityPage } from "../activity";

type Props = { params: Promise<{ username: string; filter: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { username, filter } = await params;
  return { title: decodeURIComponent(username), robots: filter === "bookmarks" ? { index: false } : undefined };
}

export default async function Page({ params }: Props) {
  const { username, filter } = await params;
  return <ActivityPage username={decodeURIComponent(username)} filter={filter} />;
}
