import type { Metadata } from "next";
import { CategoriesPage } from "@/components/categories-page";
import { firstParam, ListPage, parsePage } from "@/components/list-page";
import { getSite } from "@/lib/server-api";

export async function generateMetadata(): Promise<Metadata> {
  const site = await getSite();
  const title = site.settings.title || "Casnode";
  return {
    title: { absolute: site.settings.short_site_description ? `${title} - ${site.settings.short_site_description}` : title },
    alternates: { canonical: "/" },
  };
}

// The home page is the first item of top_menu, like Discourse.
export default async function Home({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const site = await getSite();
  const params = await searchParams;
  const first = (site.settings.top_menu || "latest").split("|")[0];
  if (first === "categories") {
    return <CategoriesPage />;
  }
  const filter = (first === "new" || first === "unread") && !site.currentUser ? "latest" : first;
  return <ListPage filter={filter} page={parsePage(params.page)} period={firstParam(params.period)} />;
}
