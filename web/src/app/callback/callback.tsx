"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState } from "react";
import { useSite } from "@/components/site-provider";
import { apiPost, errorMessage } from "@/lib/api";

// Casdoor sends the user back here with ?code=; the backend trades it for the account and signs the user in.
function CallbackInner() {
  const params = useSearchParams();
  const router = useRouter();
  const { t, refreshSite } = useSite();
  const [failure, setFailure] = useState("");
  const started = useRef(false);
  const code = params.get("code");
  const error = failure || (code ? "" : params.get("error_description") || params.get("error") || "missing code");

  useEffect(() => {
    if (started.current) {
      return;
    }
    started.current = true;
    if (!code) {
      return;
    }
    apiPost("signin", { code, state: params.get("state") ?? "" })
      .then(async () => {
        let returnTo = "/";
        try {
          returnTo = sessionStorage.getItem("casnode-return-to") || "/";
          sessionStorage.removeItem("casnode-return-to");
        } catch {
          // back to the home page
        }
        await refreshSite().catch(() => undefined);
        router.replace(returnTo.startsWith("/") && !returnTo.startsWith("//") ? returnTo : "/");
        router.refresh();
      })
      .catch((err) => setFailure(errorMessage(err)));
  }, [code, params, router, refreshSite]);

  return (
    <div className="login-page">
      <h1>{error ? t("login.failed") : t("login.signingIn")}</h1>
      {error ? <div className="alert alert-error">{error}</div> : <div className="spinner" />}
    </div>
  );
}

export function Callback() {
  return (
    <Suspense fallback={<div className="spinner" />}>
      <CallbackInner />
    </Suspense>
  );
}
