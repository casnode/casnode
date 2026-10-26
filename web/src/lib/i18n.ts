import en, { type MessageKey } from "@/lib/locales/en";
import zh from "@/lib/locales/zh";

export type { MessageKey };
export type Locale = "en" | "zh";
export const locales: { value: Locale; label: string }[] = [
  { value: "en", label: "English" },
  { value: "zh", label: "简体中文" },
];

const dictionaries: Record<Locale, Partial<Record<MessageKey, string>>> = { en, zh };

export function normalizeLocale(value?: string | null): Locale | null {
  if (!value) {
    return null;
  }
  if (value.toLowerCase().startsWith("zh")) {
    return "zh";
  }
  if (value.toLowerCase().startsWith("en")) {
    return "en";
  }
  return null;
}

export type Translate = (key: MessageKey, vars?: Record<string, string | number>) => string;

export function makeT(locale: Locale): Translate {
  const dict = dictionaries[locale] ?? en;
  return (key, vars) => {
    let text: string = dict[key] ?? en[key] ?? key;
    if (vars) {
      for (const [name, value] of Object.entries(vars)) {
        text = text.replaceAll(`{${name}}`, String(value));
      }
    }
    return text;
  };
}
