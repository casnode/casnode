import Script from "next/script";

// Google Analytics and Baidu Tongji, when the admins set their IDs on /admin/site_settings.
export function Analytics({ googleId, baiduId }: { googleId?: string; baiduId?: string }) {
  const safe = (id?: string) => (id && /^[\w-]+$/.test(id) ? id : "");
  const ga = safe(googleId);
  const baidu = safe(baiduId);
  return (
    <>
      {ga ? (
        <>
          <Script src={`https://www.googletagmanager.com/gtag/js?id=${ga}`} strategy="afterInteractive" />
          <Script id="ga" strategy="afterInteractive">
            {`window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}gtag('js',new Date());gtag('config','${ga}');`}
          </Script>
        </>
      ) : null}
      {baidu ? (
        <Script id="baidu" strategy="afterInteractive">
          {`var _hmt=_hmt||[];(function(){var hm=document.createElement("script");hm.src="https://hm.baidu.com/hm.js?${baidu}";var s=document.getElementsByTagName("script")[0];s.parentNode.insertBefore(hm,s);})();`}
        </Script>
      ) : null}
    </>
  );
}
