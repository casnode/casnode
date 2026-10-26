import { notFound } from "next/navigation";
import { Cooked } from "@/components/cooked";
import { serverGet } from "@/lib/server-api";

// The FAQ (community guidelines), terms of service and privacy policy, written by the admins in the site settings.
export async function StaticPage({ name, title }: { name: string; title: string }) {
  const res = await serverGet<string>("get-static-page", { name });
  if (res.status !== "ok" || (!res.data && name !== "faq")) {
    notFound();
  }
  return (
    <div className="about-page">
      <h1 className="page-title">{title}</h1>
      <Cooked html={res.data} className="p-0" />
    </div>
  );
}
