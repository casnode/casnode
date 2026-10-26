"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { CategoryBadge } from "@/components/category-badge";
import { useSite } from "@/components/site-provider";
import { apiPost, errorMessage } from "@/lib/api";
import type { Category } from "@/lib/types";
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
  const set = (patch: Partial<Category>) => setForm((f) => ({ ...f, ...patch }));

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
