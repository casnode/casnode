import type { Metadata } from "next";
import { ForgotPasswordForm } from "./password-reset-form";

export const metadata: Metadata = { title: "Reset Password", robots: { index: false } };

export default function Page() {
  return <ForgotPasswordForm />;
}
