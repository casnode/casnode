import type { Metadata } from "next";
import { MessagesPage } from "./messages";

export const metadata: Metadata = { robots: { index: false } };

export default async function Page({ params }: { params: Promise<{ username: string }> }) {
  return <MessagesPage username={decodeURIComponent((await params).username)} box="inbox" />;
}
