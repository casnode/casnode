import { cache } from "react";
import { serverGet } from "@/lib/server-api";
import type { Group } from "@/lib/types";

export const loadGroup = cache(async (name: string) => {
  const res = await serverGet<Group>("get-group", { name });
  return res.status === "ok" ? res.data : null;
});
