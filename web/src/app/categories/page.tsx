import type { Metadata } from "next";
import { CategoriesPage } from "@/components/categories-page";
import { getT } from "@/lib/server-i18n";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("nav.categories"), alternates: { canonical: "/categories" } };
}

export default function Page() {
  return <CategoriesPage />;
}
