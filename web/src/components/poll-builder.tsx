"use client";

import { useState } from "react";
import { Icon } from "@/components/icon";
import { useSite } from "@/components/site-provider";
import { Dialog, DialogContent } from "@/components/ui/dialog";

// The composer's "Build Poll" dialog: it writes the [poll] block, like Discourse's poll builder.
export function PollBuilder({ onClose, onInsert }: { onClose: () => void; onInsert: (markdown: string) => void }) {
  const { t } = useSite();
  const [type, setType] = useState<"regular" | "multiple">("regular");
  const [options, setOptions] = useState("");
  const [min, setMin] = useState(1);
  const [max, setMax] = useState(2);
  const [results, setResults] = useState("always");
  const [isPublic, setIsPublic] = useState(false);
  const [close, setClose] = useState("");

  const lines = options
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);

  const build = () => {
    const attrs = [`type=${type}`];
    if (type === "multiple") {
      attrs.push(`min=${Math.max(1, min)}`, `max=${Math.max(min, Math.min(max, lines.length))}`);
    }
    if (results !== "always") {
      attrs.push(`results=${results}`);
    }
    if (isPublic) {
      attrs.push("public=true");
    }
    if (close) {
      attrs.push(`close=${new Date(close).toISOString()}`);
    }
    return `\n[poll ${attrs.join(" ")}]\n${lines.map((line) => `* ${line}`).join("\n")}\n[/poll]\n`;
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        title={t("poll.build")}
        className="poll-builder"
        footer={
          <>
            <button type="button" className="btn btn-primary" disabled={lines.length < 2} onClick={() => onInsert(build())}>
              <Icon name="chart-bar" />
              {t("poll.insert")}
            </button>
            <button type="button" className="btn btn-flat" onClick={onClose}>
              {t("common.cancel")}
            </button>
          </>
        }
      >
        <div className="control-group">
          <label className="form-label">{t("poll.type")}</label>
          <div className="flex gap-2">
            <button type="button" className={`btn btn-small ${type === "regular" ? "btn-primary" : "btn-default"}`} onClick={() => setType("regular")}>
              {t("poll.typeRegular")}
            </button>
            <button type="button" className={`btn btn-small ${type === "multiple" ? "btn-primary" : "btn-default"}`} onClick={() => setType("multiple")}>
              {t("poll.typeMultiple")}
            </button>
          </div>
        </div>
        <div className="control-group">
          <label className="form-label">{t("poll.options", { count: lines.length })}</label>
          <textarea className="d-input" rows={6} value={options} placeholder={t("poll.optionsPlaceholder")} onChange={(e) => setOptions(e.target.value)} />
        </div>
        {type === "multiple" ? (
          <div className="control-group flex gap-4">
            <label>
              {t("poll.min")} <input className="d-input inline-block w-[5em]" type="number" min={1} value={min} onChange={(e) => setMin(Number(e.target.value))} />
            </label>
            <label>
              {t("poll.max")} <input className="d-input inline-block w-[5em]" type="number" min={1} value={max} onChange={(e) => setMax(Number(e.target.value))} />
            </label>
          </div>
        ) : null}
        <div className="control-group">
          <label className="form-label">{t("poll.resultsLabel")}</label>
          <select className="d-select" value={results} onChange={(e) => setResults(e.target.value)}>
            <option value="always">{t("poll.resultsAlways")}</option>
            <option value="on_vote">{t("poll.resultsVote")}</option>
            <option value="on_close">{t("poll.resultsClose")}</option>
            <option value="staff_only">{t("poll.resultsStaff")}</option>
          </select>
        </div>
        <div className="control-group">
          <label className="form-label">{t("poll.autoClose")}</label>
          <input className="d-input max-w-[16em]" type="datetime-local" value={close} onChange={(e) => setClose(e.target.value)} />
        </div>
        <label className="form-check">
          <input type="checkbox" checked={isPublic} onChange={(e) => setIsPublic(e.target.checked)} />
          {t("poll.publicLabel")}
        </label>
      </DialogContent>
    </Dialog>
  );
}
