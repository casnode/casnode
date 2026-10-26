import type { Metadata } from "next";
import { firstParam, ListPage, parsePage } from "@/components/list-page";
import { getT } from "@/lib/server-i18n";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("nav.new"), alternates: { canonical: "/new" }, robots: { index: false } };
}

export default async function Page({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const params = await searchParams;
  return <ListPage filter="new" page={parsePage(params.page)} period={firstParam(params.period)} order={firstParam(params.order)} ascending={firstParam(params.ascending) === "true"} />;
}
