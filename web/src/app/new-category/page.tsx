import type { Metadata } from "next";
import { CategoryForm } from "@/components/category-form";

export const metadata: Metadata = { title: "New Category", robots: { index: false } };

export default function Page() {
  return <CategoryForm />;
}
