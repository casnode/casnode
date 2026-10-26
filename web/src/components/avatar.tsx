import type { User } from "@/lib/types";

// Discourse's letter avatars: the first letter of the name on a color picked from the name.
const letterColors = [
  "#b19c9b", "#e47c90", "#c67d28", "#a87137", "#9f7d24", "#7e9a26", "#45a33c", "#33a37b",
  "#2e9ba0", "#4c9ad1", "#8c86d6", "#b07ad4", "#d47ab0", "#e55c5c", "#f0a830", "#5f9e5f",
  "#3fa6c2", "#7b8bd9", "#c27c3e", "#8f9f4f", "#d4704f", "#6f8faf", "#a07fbf", "#cf6f8f",
];

function letterColor(name: string): string {
  let hash = 0;
  for (const ch of name) {
    hash = (hash * 31 + ch.codePointAt(0)!) | 0;
  }
  return letterColors[Math.abs(hash) % letterColors.length];
}

export function Avatar({ user, size = 45, className, title }: { user?: User | null; size?: number; className?: string; title?: string }) {
  const name = user?.username ?? "?";
  const style = { width: size, height: size };
  if (user?.avatar) {
    // eslint-disable-next-line @next/next/no-img-element
    return <img src={user.avatar} alt="" title={title ?? name} className={`avatar${className ? ` ${className}` : ""}`} style={style} loading="lazy" width={size} height={size} />;
  }
  const letter = Array.from(name)[0]?.toUpperCase() ?? "?";
  return (
    <span
      className={`letter-avatar${className ? ` ${className}` : ""}`}
      style={{ ...style, background: letterColor(name), fontSize: Math.round(size * 0.5) }}
      title={title ?? name}
      aria-label={name}
      role="img"
    >
      {letter}
    </span>
  );
}
