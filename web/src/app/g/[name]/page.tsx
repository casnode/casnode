import { notFound } from "next/navigation";
import { GroupMembers } from "@/components/group/group";
import { loadGroup } from "./group-data";

export default async function Page({ params }: { params: Promise<{ name: string }> }) {
  const group = await loadGroup(decodeURIComponent((await params).name));
  if (!group) {
    notFound();
  }
  return <GroupMembers group={group} />;
}
