import { cache } from "react";
import { serverGet } from "@/lib/server-api";
import type { User, UserSummary } from "@/lib/types";

export const loadUser = cache(async (username: string) => {
  const res = await serverGet<User>("get-user", { username });
  return res.status === "ok" ? res.data : null;
});

export const loadSummary = cache(async (username: string) => {
  const res = await serverGet<User, UserSummary>("get-user", { username, summary: "true" });
  return res.status === "ok" ? res.data2 : null;
});
