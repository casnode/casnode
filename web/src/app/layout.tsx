import "@fontsource-variable/inter";
import type { Metadata, Viewport } from "next";
import { Toaster } from "sonner";
import { Analytics } from "@/components/analytics";
import { BrandStyle } from "@/components/brand-style";
import { Composer } from "@/components/composer";
import { ComposerProvider } from "@/components/composer-provider";
import { SiteHeader } from "@/components/header";
import { LayoutStateProvider } from "@/components/layout-state";
import { MainOutlet } from "@/components/main-outlet";
import { SiteProvider } from "@/components/site-provider";
import { getOrigin, getSite } from "@/lib/server-api";
import { getLocale } from "@/lib/server-i18n";
import "./globals.css";

export async function generateMetadata(): Promise<Metadata> {
  const site = await getSite();
  const title = site.settings.title || "Casnode";
  const origin = await getOrigin();
  return {
    metadataBase: new URL(origin),
    title: { default: title, template: `%s - ${title}` },
    description: site.settings.site_description,
    applicationName: title,
    icons: { icon: site.settings.favicon_url || "/favicon.svg" },
    alternates: { types: { "application/rss+xml": `${origin}/latest.rss` } },
    openGraph: { siteName: title, type: "website", images: site.settings.logo_url ? [site.settings.logo_url] : undefined },
    formatDetection: { telephone: false },
  };
}

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ffffff" },
    { media: "(prefers-color-scheme: dark)", color: "#111111" },
  ],
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const site = await getSite();
  const locale = await getLocale();
  return (
    <html lang={locale === "zh" ? "zh-CN" : "en"} suppressHydrationWarning>
      <body>
        <SiteProvider initialSite={site} locale={locale}>
          <BrandStyle />
          <LayoutStateProvider>
            <ComposerProvider>
              {site.settings.header_html ? <div dangerouslySetInnerHTML={{ __html: site.settings.header_html }} /> : null}
              <SiteHeader />
              <MainOutlet>{children}</MainOutlet>
              <Composer />
              <Toaster position="top-center" richColors closeButton />
            </ComposerProvider>
          </LayoutStateProvider>
        </SiteProvider>
        <Analytics googleId={site.settings.google_analytics_id} baiduId={site.settings.baidu_analytics_id} />
      </body>
    </html>
  );
}
