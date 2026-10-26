"use client";

import { useSite } from "@/components/site-provider";

const fonts: Record<string, string> = {
  inter: `"Inter Variable", Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", "Noto Sans CJK SC", Roboto, sans-serif`,
  system: `system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", "Noto Sans CJK SC", sans-serif`,
  helvetica: `"Helvetica Neue", Helvetica, Arial, "PingFang SC", "Microsoft YaHei", sans-serif`,
  arial: `Arial, "Helvetica Neue", Helvetica, "PingFang SC", "Microsoft YaHei", sans-serif`,
  verdana: `Verdana, Geneva, "PingFang SC", "Microsoft YaHei", sans-serif`,
  georgia: `Georgia, "Times New Roman", "Songti SC", SimSun, serif`,
  times: `"Times New Roman", Times, "Songti SC", SimSun, serif`,
};

const isColor = (value: string | undefined) => !!value && /^#([0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$/i.test(value);

// The variables a color scheme sets, for the light or the dark one. A dark setting left empty takes the light one.
function schemeVars(settings: Record<string, string>, dark: boolean) {
  const pick = (name: string) => {
    const value = dark && isColor(settings[`${name}_dark`]) ? settings[`${name}_dark`] : settings[name];
    return isColor(value) ? value : "";
  };
  const vars: string[] = [];
  const brand = pick("brand_color");
  if (brand) {
    const mix = (percent: number) => `color-mix(in srgb, var(--tertiary) ${percent}%, var(--secondary))`;
    vars.push(
      `--tertiary: ${brand}`,
      `--tertiary-50: ${mix(6)}`,
      `--tertiary-very-low: ${mix(12)}`,
      `--tertiary-low: ${mix(18)}`,
      `--tertiary-medium: ${mix(60)}`,
      `--tertiary-high: ${mix(85)}`,
      `--tertiary-hover: color-mix(in srgb, var(--tertiary) 75%, ${dark ? "white" : "black"})`,
    );
  }
  const accent = pick("accent_color");
  if (accent) {
    vars.push(`--quaternary: ${accent}`, `--quaternary-low: color-mix(in srgb, var(--quaternary) 25%, var(--secondary))`);
  }
  const headerBackground = pick("header_background_color");
  const headerText = pick("header_text_color");
  if (headerBackground) {
    vars.push(`--header_background: ${headerBackground}`);
  }
  if (headerText) {
    vars.push(`--header_primary: ${headerText}`);
  }
  if (headerBackground || headerText) {
    const mix = (percent: number) => `color-mix(in srgb, var(--header_primary) ${percent}%, var(--header_background))`;
    vars.push(`--header_primary-low-mid: ${mix(23)}`, `--header_primary-medium: ${mix(39)}`, `--header_primary-high: ${mix(53)}`);
  }
  return vars;
}

// The branding settings (colors, fonts, custom CSS) as a style element, so a saved setting shows without a reload.
export function BrandStyle() {
  const { settings } = useSite();
  const rules: string[] = [];

  const light = schemeVars(settings, false);
  const dark = schemeVars(settings, true);
  if (light.length) {
    rules.push(`:root { ${light.join("; ")}; }`);
  }
  if (dark.length) {
    rules.push(`@media (prefers-color-scheme: dark) { :root:not([data-theme="light"]) { ${dark.join("; ")}; } }`);
  }

  const baseFont = fonts[settings.base_font] ? settings.base_font : "inter";
  const headingFont = fonts[settings.heading_font] ? settings.heading_font : "inter";
  if (baseFont !== "inter") {
    rules.push(`:root { --font-family: ${fonts[baseFont]}; }`);
  }
  if (headingFont !== baseFont) {
    rules.push(`h1, h2, h3, h4, h5, h6, .d-header .site-title-text, .homepage-banner .banner-title { font-family: ${fonts[headingFont]}; }`);
  }

  if (settings.custom_css) {
    rules.push(settings.custom_css);
  }
  if (!rules.length) {
    return null;
  }
  const css = rules.join("\n").replace(/<\/style/gi, "<\\/style");
  return <style id="brand-style" dangerouslySetInnerHTML={{ __html: css }} />;
}
