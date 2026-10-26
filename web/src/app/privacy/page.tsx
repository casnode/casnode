import type { Metadata } from "next";
import { StaticPage } from "@/components/static-page";
import { getT } from "@/lib/server-i18n";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("about.privacy"), alternates: { canonical: "/privacy" } };
}

export default async function Page() {
  const t = await getT();
  return <StaticPage name="privacy" title={t("about.privacy")} />;
}
