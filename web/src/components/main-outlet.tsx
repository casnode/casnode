"use client";

import type { ReactNode } from "react";
import { useLayoutState } from "@/components/layout-state";
import { Sidebar } from "@/components/sidebar";
import { useSite } from "@/components/site-provider";

export function MainOutlet({ children }: { children: ReactNode }) {
  const { sidebarVisible } = useLayoutState();
  const { settings } = useSite();
  return (
    <div id="main-outlet-wrapper" className={sidebarVisible ? "" : "no-sidebar"}>
      <Sidebar />
      <main id="main-outlet">
        {children}
        {settings.footer_html ? <div className="footer-html" dangerouslySetInnerHTML={{ __html: settings.footer_html }} /> : null}
      </main>
    </div>
  );
}
