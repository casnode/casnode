import type { Metadata } from "next";
import { StaticPage } from "@/components/static-page";
import { getT } from "@/lib/server-i18n";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("about.faq"), alternates: { canonical: "/faq" } };
}

export default async function Page() {
  const t = await getT();
  return <StaticPage name="faq" title={t("about.faq")} />;
}
