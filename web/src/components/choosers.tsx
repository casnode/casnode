"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Avatar } from "@/components/avatar";
import { CategoryBadge, plainDescription } from "@/components/category-badge";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { apiGet } from "@/lib/api";
import type { Category, Tag, User } from "@/lib/types";

// The category drop-down with a filter box, like Discourse's category-chooser.
export function CategoryChooser({
  value,
  onChange,
  placeholder,
  filter,
  className,
  allowNone,
  noneLabel,
}: {
  value: number;
  onChange: (id: number) => void;
  placeholder: string;
  filter?: (c: Category) => boolean;
  className?: string;
  allowNone?: boolean;
  noneLabel?: string;
}) {
  const { categories } = useSite();
  const [open, setOpen] = useState(false);
  const [term, setTerm] = useState("");
  const [highlighted, setHighlighted] = useState(0);
  const selected = categories.find((c) => c.id === value);

  const options = useMemo(() => {
    const q = term.trim().toLowerCase();
    return categories.filter((c) => (!filter || filter(c)) && (!q || c.name.toLowerCase().includes(q) || c.slug.includes(q)));
  }, [categories, term, filter]);

  const choose = (id: number) => {
    onChange(id);
    setOpen(false);
    setTerm("");
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button type="button" className={`select-kit-header category-chooser ${className ?? ""}`} aria-label={placeholder}>
          <span className="selected-name">{selected ? <CategoryBadge category={selected} categories={categories} link={false} /> : <span>{allowNone && noneLabel ? noneLabel : placeholder}</span>}</span>
          <Icon name="caret-down" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-[22em] p-0" onOpenAutoFocus={(e) => e.preventDefault()}>
        <div className="dropdown-filter">
          <input
            autoFocus
            value={term}
            placeholder={placeholder}
            onChange={(e) => {
              setTerm(e.target.value);
              setHighlighted(0);
            }}
            onKeyDown={(e) => {
              if (e.key === "ArrowDown") {
                e.preventDefault();
                setHighlighted((h) => Math.min(h + 1, options.length - 1));
              } else if (e.key === "ArrowUp") {
                e.preventDefault();
                setHighlighted((h) => Math.max(h - 1, 0));
              } else if (e.key === "Enter" && options[highlighted]) {
                e.preventDefault();
                choose(options[highlighted].id);
              }
            }}
          />
        </div>
        <div className="max-h-[22em] overflow-y-auto py-1">
          {allowNone ? (
            <button type="button" className={`dropdown-menu-item${value === 0 ? " is-selected" : ""}`} onClick={() => choose(0)}>
              <span className="item-text">
                <span>{noneLabel}</span>
              </span>
            </button>
          ) : null}
          {options.map((c, i) => (
            <button
              key={c.id}
              type="button"
              className={`dropdown-menu-item${c.id === value ? " is-selected" : ""}${i === highlighted ? " is-highlighted" : ""}`}
              onMouseEnter={() => setHighlighted(i)}
              onClick={() => choose(c.id)}
            >
              <span className="item-text">
                <span className="flex items-center gap-2">
                  <CategoryBadge category={c} categories={categories} link={false} />
                  <span className="text-[var(--primary-medium)] text-[0.8706em]">× {c.topicCount}</span>
                </span>
                {c.description ? <span className="category-row-description">{plainDescription(c.description)}</span> : null}
              </span>
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

// A box of chips with suggestions while typing: tags or users.
function ChipsInput<T>({
  values,
  onChange,
  placeholder,
  search,
  renderOption,
  toValue,
  allowCreate,
  className,
  max,
}: {
  values: string[];
  onChange: (values: string[]) => void;
  placeholder: string;
  search: (term: string) => Promise<T[]>;
  renderOption: (item: T) => React.ReactNode;
  toValue: (item: T) => string;
  allowCreate?: boolean;
  className: string;
  max?: number;
}) {
  const [term, setTerm] = useState("");
  const [options, setOptions] = useState<T[]>([]);
  const [highlighted, setHighlighted] = useState(0);
  const [focused, setFocused] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!focused) {
      return;
    }
    let cancelled = false;
    const timer = setTimeout(() => {
      search(term.trim())
        .then((items) => {
          if (!cancelled) {
            setOptions(items.filter((item) => !values.includes(toValue(item))));
            setHighlighted(0);
          }
        })
        .catch(() => undefined);
    }, 200);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [term, focused, search, values, toValue]);

  const add = (value: string) => {
    const v = value.trim();
    if (!v || values.includes(v) || (max && values.length >= max)) {
      return;
    }
    onChange([...values, v]);
    setTerm("");
  };

  return (
    <div className={className} onClick={() => inputRef.current?.focus()}>
      {values.map((v) => (
        <button key={v} type="button" className="selected-chip" onClick={() => onChange(values.filter((x) => x !== v))}>
          {v}
          <Icon name="xmark" />
        </button>
      ))}
      <input
        ref={inputRef}
        value={term}
        placeholder={values.length === 0 ? placeholder : ""}
        onFocus={() => setFocused(true)}
        onBlur={() => setTimeout(() => setFocused(false), 150)}
        onChange={(e) => {
          // a comma (typed or pasted) ends a value, like Enter
          const parts = e.target.value.split(/[,，]/);
          if (allowCreate && parts.length > 1) {
            const added = parts.slice(0, -1).map((p) => p.trim()).filter((p) => p && !values.includes(p));
            const room = max ? Math.max(0, max - values.length) : added.length;
            onChange([...values, ...Array.from(new Set(added)).slice(0, room)]);
            setTerm(parts[parts.length - 1]);
            return;
          }
          setTerm(e.target.value);
        }}
        onKeyDown={(e) => {
          if (e.key === "ArrowDown") {
            e.preventDefault();
            setHighlighted((h) => Math.min(h + 1, options.length - 1));
          } else if (e.key === "ArrowUp") {
            e.preventDefault();
            setHighlighted((h) => Math.max(h - 1, 0));
          } else if (e.key === "Enter" || e.key === "," || e.key === "Tab") {
            if (options[highlighted] && (e.key !== "Tab" || term)) {
              e.preventDefault();
              add(toValue(options[highlighted]));
            } else if (allowCreate && term.trim()) {
              e.preventDefault();
              add(term);
            }
          } else if (e.key === "Backspace" && !term && values.length > 0) {
            onChange(values.slice(0, -1));
          }
        }}
      />
      {focused && options.length > 0 ? (
        <ul className="mention-autocomplete" style={{ top: "100%", left: 0 }}>
          {options.slice(0, 8).map((item, i) => (
            <li key={toValue(item)}>
              <button
                type="button"
                className={i === highlighted ? "selected" : ""}
                onMouseDown={(e) => {
                  e.preventDefault();
                  add(toValue(item));
                }}
              >
                {renderOption(item)}
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

const searchTags = async (term: string) => (await apiGet<Tag[]>("get-tags", { q: term })).data ?? [];
const tagValue = (tag: Tag) => tag.name;
const searchUsers = async (term: string) => (await apiGet<User[]>("search-users", { term })).data ?? [];
const userValue = (user: User) => user.username;

export function TagChooser({ values, onChange, placeholder, max }: { values: string[]; onChange: (v: string[]) => void; placeholder: string; max?: number }) {
  return (
    <ChipsInput
      className="tags-input"
      values={values}
      onChange={onChange}
      placeholder={placeholder}
      search={searchTags}
      toValue={tagValue}
      allowCreate
      max={max}
      renderOption={(tag) => (
        <>
          <Icon name="tag" className="text-[var(--primary-medium)]" />
          <span>{tag.name}</span>
          <span className="ml-auto text-[var(--primary-medium)] text-[0.8706em]">× {tag.topicCount}</span>
        </>
      )}
    />
  );
}

export function UserChooser({ values, onChange, placeholder }: { values: string[]; onChange: (v: string[]) => void; placeholder: string }) {
  return (
    <ChipsInput
      className="recipients-input"
      values={values}
      onChange={onChange}
      placeholder={placeholder}
      search={searchUsers}
      toValue={userValue}
      renderOption={(user) => (
        <>
          <Avatar user={user} size={20} />
          <span className="font-bold">{user.username}</span>
          {user.name && user.name !== user.username ? <span className="text-[var(--primary-medium)]">{user.name}</span> : null}
        </>
      )}
    />
  );
}
