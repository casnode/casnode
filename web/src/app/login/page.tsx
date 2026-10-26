import type { Metadata } from "next";
import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "Log In", robots: { index: false } };

export default function Page() {
  return <LoginForm />;
}
