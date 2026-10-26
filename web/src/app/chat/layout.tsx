import type { Metadata } from "next";
import { ChatShell } from "@/components/chat/chat";
import { getSite } from "@/lib/server-api";
import { getT } from "@/lib/server-i18n";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: t("chat.title"), robots: { index: false } };
}

export default async function ChatLayout({ children }: { children: React.ReactNode }) {
  const site = await getSite();
  const t = await getT();
  if (site.settings.chat_enabled === "false") {
    return <div className="alert alert-info">{t("chat.disabled")}</div>;
  }
  if (!site.currentUser) {
    return <div className="alert alert-info">{t("login.required")}</div>;
  }
  return <ChatShell>{children}</ChatShell>;
}
