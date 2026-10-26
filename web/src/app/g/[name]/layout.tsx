import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { GroupHeader, GroupNav } from "@/components/group/group";
import { loadGroup } from "./group-data";

type Props = { params: Promise<{ name: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const group = await loadGroup(decodeURIComponent((await params).name));
  return group ? { title: group.fullName || group.name } : {};
}

export default async function GroupLayout({ children, params }: Props & { children: React.ReactNode }) {
  const group = await loadGroup(decodeURIComponent((await params).name));
  if (!group) {
    notFound();
  }
  return (
    <div className="group-page">
      <GroupHeader group={group} />
      <GroupNav group={group} />
      <section className="group-content">{children}</section>
    </div>
  );
}
