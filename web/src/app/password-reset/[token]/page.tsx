import type { Metadata } from "next";
import { NewPasswordForm } from "../password-reset-form";

export const metadata: Metadata = { title: "Reset Password", robots: { index: false }, referrer: "no-referrer" };

export default async function Page({ params }: { params: Promise<{ token: string }> }) {
  const { token } = await params;
  return <NewPasswordForm token={token} />;
}
