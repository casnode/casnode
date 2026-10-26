import type { Metadata } from "next";
import { ActivityPage } from "./activity";

type Props = { params: Promise<{ username: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const username = decodeURIComponent((await params).username);
  return { title: username };
}

export default async function Page({ params }: Props) {
  return <ActivityPage username={decodeURIComponent((await params).username)} filter="" />;
}
