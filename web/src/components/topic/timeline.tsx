"use client";

import { useRef, useState } from "react";
import { useSite } from "@/components/site-provider";
import { shortDate } from "@/lib/format";
import { isValidDate } from "@/lib/utils";

const scrollAreaHeight = 300;
const handleHeight = 50;

// The scroll bar on the right of a topic: where you are ("4 / 22" and the date), and drag or click to jump.
export function Timeline({
  total,
  current,
  currentDate,
  firstDate,
  lastDate,
  onJump,
  children,
}: {
  total: number;
  current: number;
  currentDate?: string;
  firstDate: string;
  lastDate: string;
  onJump: (index: number) => void;
  children?: React.ReactNode;
}) {
  const { locale, t } = useSite();
  const areaRef = useRef<HTMLDivElement>(null);
  const [dragIndex, setDragIndex] = useState<number | null>(null);
  const shown = dragIndex ?? current;
  const range = scrollAreaHeight - handleHeight;
  const top = total > 1 ? (shown / (total - 1)) * range : 0;

  const indexAt = (clientY: number) => {
    const rect = areaRef.current!.getBoundingClientRect();
    const y = Math.min(range, Math.max(0, clientY - rect.top - handleHeight / 2));
    return total > 1 ? Math.round((y / range) * (total - 1)) : 0;
  };

  const startDrag = (e: React.PointerEvent) => {
    e.preventDefault();
    const move = (ev: PointerEvent) => setDragIndex(indexAt(ev.clientY));
    const up = (ev: PointerEvent) => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      const index = indexAt(ev.clientY);
      setDragIndex(null);
      onJump(index);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  };

  const fmt = (value?: string) => (isValidDate(value) ? shortDate(new Date(value!), locale, new Date(value!).getFullYear() !== new Date().getFullYear()) : "");

  return (
    <div className="timeline-container">
      <div className="topic-timeline">
        <div className="timeline-date-wrapper">
          <button type="button" className="start-date" onClick={() => onJump(0)} title={t("topic.jumpToTop")}>
            {fmt(firstDate)}
          </button>
        </div>
        <div
          className="timeline-scrollarea"
          ref={areaRef}
          style={{ height: scrollAreaHeight }}
          onClick={(e) => {
            if (e.target === e.currentTarget) {
              onJump(indexAt(e.clientY));
            }
          }}
        >
          <div className="timeline-padding" style={{ height: top }} />
          <div className={`timeline-scroller${dragIndex !== null ? " dragging" : ""}`} onPointerDown={startDrag}>
            <div className="timeline-handle" />
            <div className="timeline-scroller-content">
              <div className="timeline-replies">
                {Math.min(shown + 1, total)} / {total}
              </div>
              <div className="timeline-ago">{fmt(dragIndex === null ? currentDate : undefined)}</div>
            </div>
          </div>
        </div>
        <div className="timeline-date-wrapper">
          <button type="button" className="now-date" onClick={() => onJump(total - 1)} title={t("topic.jumpToBottom")}>
            {fmt(lastDate)}
          </button>
        </div>
        {children ? <div className="timeline-footer-controls">{children}</div> : null}
      </div>
    </div>
  );
}
