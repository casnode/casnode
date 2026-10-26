import type { Metadata } from "next";
import { Callback } from "./callback";

export const metadata: Metadata = { robots: { index: false } };

export default function Page() {
  return <Callback />;
}
