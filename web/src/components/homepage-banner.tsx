"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";

// The welcome banner on the home page, with a search box, like Discourse's search banner.
export function HomepageBanner() {
  const { settings, t } = useSite();
  const router = useRouter();
  const [term, setTerm] = useState("");
  const title = settings.homepage_banner_title;
  const text = settings.homepage_banner_text;
  if (!title && !text) {
    return null;
  }

  const image = settings.homepage_banner_image_url;
  return (
    <div className={`homepage-banner${image ? " has-image" : ""}`} style={image ? { backgroundImage: `url(${JSON.stringify(image)})` } : undefined}>
      {title ? <h1 className="banner-title">{title}</h1> : null}
      {text ? <p className="banner-text">{text}</p> : null}
      {settings.homepage_banner_search !== "false" ? (
        <form
          className="banner-search"
          role="search"
          onSubmit={(e) => {
            e.preventDefault();
            if (term.trim()) {
              router.push(`/search?q=${encodeURIComponent(term.trim())}`);
            }
          }}
        >
          <input className="d-input" type="search" value={term} placeholder={t("search.placeholder")} aria-label={t("search.placeholder")} onChange={(e) => setTerm(e.target.value)} />
          <button type="submit" className="btn btn-primary" aria-label={t("search.button")}>
            <Icon name="search" />
          </button>
        </form>
      ) : null}
    </div>
  );
}
