"use client";

import { useComposer } from "@/components/composer-provider";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";

// "Message" on a profile writes to that user; without a user name it is the "New Message" of the inbox.
export function MessageButton({ username }: { username?: string }) {
  const { t, settings } = useSite();
  const { open } = useComposer();
  if (settings.enable_personal_messages === "false") {
    return null;
  }
  return (
    <button type="button" className="btn btn-primary" onClick={() => open({ action: "privateMessage", recipients: username ? [username] : [] })}>
      <Icon name="envelope" />
      {username ? t("user.message") : t("sidebar.newMessage")}
    </button>
  );
}
