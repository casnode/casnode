import type { Metadata } from "next";
import { MessagesPage } from "../messages";

export const metadata: Metadata = { robots: { index: false } };

export default async function Page({ params }: { params: Promise<{ username: string; box: string }> }) {
  const { username, box } = await params;
  return <MessagesPage username={decodeURIComponent(username)} box={box === "sent" ? "sent" : "inbox"} />;
}
