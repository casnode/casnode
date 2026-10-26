"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { CategoryBadge } from "@/components/category-badge";
import { useSite } from "@/components/site-provider";
import { apiGet, apiPost, errorMessage } from "@/lib/api";
import type { Category, Group } from "@/lib/types";
import { categoryUrl } from "@/lib/utils";

const colors = ["0088CC", "12A89D", "25AAE2", "3AB54A", "9EB83B", "F7941D", "E45735", "BF1E2E", "ED207B", "652D90", "92278F", "8C6238", "808281", "231F20"];

// Creating and editing a category, for admins (/new-category and /c/<slug>/<id>/edit).
export function CategoryForm({ category }: { category?: Category }) {
  const { t, categories, currentUser, refreshSite } = useSite();
  const router = useRouter();
  const [form, setForm] = useState<Partial<Category>>(
    category ?? { name: "", slug: "", description: "", color: "0088CC", textColor: "FFFFFF", parentId: 0, readRestricted: false, createRestricted: false, topicTemplate: "" },
  );
  const [busy, setBusy] = useState(false);
  const [groups, setGroups] = useState<Group[]>([]);
  const set = (patch: Partial<Category>) => setForm((f) => ({ ...f, ...patch }));
  const permissions = form.permissions ?? [];

  useEffect(() => {
    apiGet<Group[]>("get-groups")
      .then((res) => setGroups(res.data ?? []))
      .catch(() => undefined);
  }, []);

  if (!currentUser?.admin) {
    return <div className="alert alert-info">{t("login.required")}</div>;
  }

  const save = async () => {
    setBusy(true);
    try {
      const res = await apiPost<Category>(category ? "update-category" : "add-category", undefined, { ...form, position: form.position ?? 0 });
      await refreshSite();
      router.push(categoryUrl(res.data, categories));
      router.refresh();
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    if (!category || !window.confirm(t("categoryForm.confirmDelete"))) {
      return;
    }
    try {
      await apiPost("delete-category", { id: category.id });
      await refreshSite();
      router.push("/categories");
      router.refresh();
    } catch (err) {
      toast.error(errorMessage(err));
    }
  };

  const preview: Category = { ...(form as Category), id: category?.id ?? 0, name: form.name || "category", parentId: 0, subcategoryIds: null };

  return (
    <div className="category-form">
      <h1 className="page-title flex items-center gap-3">
        {category ? t("categories.editCategory") : t("categories.newCategory")}
        <CategoryBadge category={preview} categories={categories} link={false} />
      </h1>
      <div className="control-group">
        <label className="form-label">{t("categoryForm.name")}</label>
        <input className="d-input" value={form.name} onChange={(e) => set({ name: e.target.value })} />
      </div>
      <div className="control-group">
        <label className="form-label">{t("categoryForm.slug")}</label>
        <input className="d-input" value={form.slug} placeholder={form.name?.toLowerCase().replace(/\s+/g, "-")} onChange={(e) => set({ slug: e.target.value })} />
      </div>
      <div className="control-group">
        <label className="form-label">{t("categoryForm.parent")}</label>
        <select className="d-select" value={form.parentId ?? 0} onChange={(e) => set({ parentId: Number(e.target.value) })}>
          <option value={0}>{t("categoryForm.noParent")}</option>
          {categories
            .filter((c) => c.parentId === 0 && c.id !== category?.id)
            .map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
        </select>
      </div>
      <div className="control-group">
        <label className="form-label">{t("categoryForm.description")}</label>
        <textarea className="d-input" rows={3} value={form.description} onChange={(e) => set({ description: e.target.value })} />
      </div>
      <div className="control-group colors">
        <div>
          <label className="form-label">{t("categoryForm.color")}</label>
          <div className="flex items-center gap-2">
            <input className="d-input w-[7em]" value={form.color} onChange={(e) => set({ color: e.target.value.replace("#", "").toUpperCase() })} />
            <span className="inline-block h-6 w-6 rounded-sm" style={{ background: `#${form.color}` }} />
          </div>
          <div className="color-picker">
            {colors.map((color) => (
              <button key={color} type="button" className={form.color === color ? "selected" : ""} style={{ background: `#${color}` }} onClick={() => set({ color })} aria-label={color} />
            ))}
          </div>
        </div>
        <div>
          <label className="form-label">{t("categoryForm.textColor")}</label>
          <div className="color-picker">
            {["FFFFFF", "000000"].map((color) => (
              <button key={color} type="button" className={form.textColor === color ? "selected" : ""} style={{ background: `#${color}`, outline: "1px solid var(--primary-low)" }} onClick={() => set({ textColor: color })} aria-label={color} />
            ))}
          </div>
        </div>
      </div>
      <div className="control-group">
        <label className="form-check">
          <input type="checkbox" checked={!!form.readRestricted} onChange={(e) => set({ readRestricted: e.target.checked })} />
          {t("categoryForm.readRestricted")}
        </label>
        <label className="form-check">
          <input type="checkbox" checked={!!form.createRestricted} onChange={(e) => set({ createRestricted: e.target.checked })} />
          {t("categoryForm.createRestricted")}
        </label>
      </div>
      {form.readRestricted || form.createRestricted ? (
        <div className="control-group category-permissions">
          <label className="form-label">{t("categoryForm.groupPermissions")}</label>
          <div className="instructions mb-2">{t("categoryForm.groupPermissionsHelp")}</div>
          {permissions.map((permission, i) => (
            <div key={permission.groupId} className="mb-1 flex items-center gap-2">
              <span className="min-w-[10em] font-bold">{groups.find((g) => g.id === permission.groupId)?.name ?? permission.groupId}</span>
              <select
                className="d-select max-w-[14em]"
                value={permission.permissionType}
                onChange={(e) => set({ permissions: permissions.map((p, j) => (j === i ? { ...p, permissionType: Number(e.target.value) } : p)) })}
              >
                <option value={1}>{t("categoryForm.permissionFull")}</option>
                <option value={2}>{t("categoryForm.permissionCreatePost")}</option>
                <option value={3}>{t("categoryForm.permissionReadonly")}</option>
              </select>
              <button type="button" className="btn btn-flat btn-small" onClick={() => set({ permissions: permissions.filter((_, j) => j !== i) })}>
                {t("group.remove")}
              </button>
            </div>
          ))}
          <select
            className="d-select max-w-[16em]"
            value=""
            onChange={(e) => e.target.value && set({ permissions: [...permissions, { groupId: Number(e.target.value), permissionType: 1 }] })}
          >
            <option value="">{t("categoryForm.addGroup")}</option>
            {groups
              .filter((g) => !permissions.some((p) => p.groupId === g.id))
              .map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name}
                </option>
              ))}
          </select>
        </div>
      ) : null}
      <div className="control-group">
        <label className="form-label">{t("categoryForm.position")}</label>
        <input className="d-input w-[7em]" type="number" value={form.position ?? 0} onChange={(e) => set({ position: Number(e.target.value) })} />
      </div>
      <div className="flex gap-2">
        <button type="button" className="btn btn-primary" disabled={busy} onClick={save}>
          {category ? t("categoryForm.save") : t("categoryForm.create")}
        </button>
        {category ? (
          <button type="button" className="btn btn-danger ml-auto" onClick={remove}>
            {t("categoryForm.delete")}
          </button>
        ) : null}
      </div>
    </div>
  );
}
