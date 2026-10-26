import { notFound } from "next/navigation";
import { ChatChannelView } from "@/components/chat/chat";

// /chat/c/<slug>/<channel id>, and /chat/c/<slug>/<channel id>/<message id> for a message
export default async function Page({ params }: { params: Promise<{ parts: string[] }> }) {
  const { parts } = await params;
  const id = Number(parts[1]);
  const messageId = Number(parts[2] ?? 0);
  if (!Number.isFinite(id) || id <= 0) {
    notFound();
  }
  return <ChatChannelView key={`${id}-${messageId}`} channelId={id} targetMessageId={Number.isFinite(messageId) ? messageId : 0} />;
}
