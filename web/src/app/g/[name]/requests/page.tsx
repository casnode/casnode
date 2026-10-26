import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { GroupRequests } from "@/components/group/group";
import { loadGroup } from "../group-data";

export const metadata: Metadata = { robots: { index: false } };

export default async function Page({ params }: { params: Promise<{ name: string }> }) {
  const group = await loadGroup(decodeURIComponent((await params).name));
  if (!group) {
    notFound();
  }
  return <GroupRequests group={group} />;
}
