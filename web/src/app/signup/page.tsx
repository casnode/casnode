import type { Metadata } from "next";
import { SignupForm } from "./signup-form";

export const metadata: Metadata = { title: "Sign Up", robots: { index: false } };

export default function Page() {
  return <SignupForm />;
}
