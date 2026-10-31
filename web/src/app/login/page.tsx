import type { Metadata } from "next";
import { Suspense } from "react";
import { LoginForm } from "./login-form";

export const metadata: Metadata = { title: "Log In", robots: { index: false } };

export default function Page() {
  return (
    <Suspense fallback={<div className="spinner" />}>
      <LoginForm />
    </Suspense>
  );
}
