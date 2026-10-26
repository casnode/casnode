import type { Metadata } from "next";
import { firstParam, ListPage, parsePage } from "@/components/list-page";
import { getT } from "@/lib/server-i18n";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("nav.hot"), alternates: { canonical: "/hot" } };
}

export default async function Page({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const params = await searchParams;
  return <ListPage filter="hot" page={parsePage(params.page)} period={firstParam(params.period)} order={firstParam(params.order)} ascending={firstParam(params.ascending) === "true"} />;
}
