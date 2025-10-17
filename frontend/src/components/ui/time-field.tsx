"use client";

import React, { useEffect, useMemo, useRef, useState } from "react";
import { cn } from "@/lib/utils";

export interface TimeFieldProps {
  value?: string; // "HH:MM" or ""
  onChange?: (value: string) => void;
  disabled?: boolean;
  className?: string;
}

function clamp(n: number, min: number, max: number) {
  return Math.max(min, Math.min(max, n));
}

function parseHHMM(v?: string): { hh: string; mm: string } {
  if (!v) return { hh: "", mm: "" };
  const m = v.match(/^(\d{1,2}):(\d{1,2})$/);
  if (!m) return { hh: "", mm: "" };
  return { hh: m[1], mm: m[2] };
}

function pad2(s: string) {
  if (s.length === 0) return "";
  return s.padStart(2, "0").slice(-2);
}

export function TimeField({ value = "", onChange, disabled, className }: TimeFieldProps) {
  const { hh: initH, mm: initM } = useMemo(() => parseHHMM(value), [value]);
  const [hh, setHh] = useState(initH);
  const [mm, setMm] = useState(initM);

  const hRef = useRef<HTMLInputElement | null>(null);
  const mRef = useRef<HTMLInputElement | null>(null);

  // Sync internal state when external value changes
  useEffect(() => {
    const { hh: h, mm: m } = parseHHMM(value);
    setHh(h);
    setMm(m);
  }, [value]);

  // Emit only when time is complete (both HH and MM provided) or fully cleared.
  const emitComplete = (h: string, m: string) => {
    if (!onChange) return;
    if (h === "" && m === "") { onChange(""); return; }
    const hOk = /^\d{1,2}$/.test(h);
    const mOk = /^\d{1,2}$/.test(m);
    if (!hOk || !mOk) return; // don't emit partial values
    const hhNum = clamp(parseInt(h, 10), 0, 23);
    const mmNum = clamp(parseInt(m, 10), 0, 59);
    onChange(`${String(hhNum).padStart(2, "0")}:${String(mmNum).padStart(2, "0")}`);
  };

  const onHChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const v = e.target.value.replace(/[^0-9]/g, "");
    if (v.length <= 2) {
      setHh(v);
      if (v.length === 2 && mm.length === 2) emitComplete(v, mm);
      if (v.length === 2) mRef.current?.focus();
    }
  };

  const onMChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const v = e.target.value.replace(/[^0-9]/g, "");
    if (v.length <= 2) {
      setMm(v);
      if (v.length === 2 && hh.length === 2) emitComplete(hh, v);
    }
  };

  const onHBlur = () => {
    if (hh === "" && mm === "") { if (onChange) onChange(""); return; }
    if (hh !== "") {
      const n = clamp(parseInt(hh || "0", 10) || 0, 0, 23);
      const p = String(n).padStart(2, "0");
      setHh(p);
      if (mm.length === 2) emitComplete(p, mm);
    }
  };

  const onMBlur = () => {
    if (hh === "" && mm === "") { if (onChange) onChange(""); return; }
    if (mm !== "") {
      const n = clamp(parseInt(mm || "0", 10) || 0, 0, 59);
      const p = String(n).padStart(2, "0");
      setMm(p);
      if (hh.length === 2) emitComplete(hh, p);
    }
  };

  const onHKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === ":") { e.preventDefault(); mRef.current?.focus(); return; }
    if (e.key === "Backspace" && hh.length === 0) { e.preventDefault(); return; }
    if (e.key === "ArrowUp" || e.key === "ArrowDown") {
      e.preventDefault();
      let n = parseInt(hh || "0", 10) || 0;
      n = e.key === "ArrowUp" ? n + 1 : n - 1;
      if (n > 23) n = 0; if (n < 0) n = 23;
      const p = String(n).padStart(2, "0");
      setHh(p);
      if (mm.length === 2) emitComplete(p, mm);
    }
    if (e.key === "ArrowRight" && (hh.length === 2 || (hRef.current && hRef.current.selectionStart === 2))) {
      e.preventDefault();
      mRef.current?.focus();
    }
  };

  const onMKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === ":") { e.preventDefault(); return; }
    if (e.key === "ArrowUp" || e.key === "ArrowDown") {
      e.preventDefault();
      let n = parseInt(mm || "0", 10) || 0;
      n = e.key === "ArrowUp" ? n + 1 : n - 1;
      if (n > 59) n = 0; if (n < 0) n = 59;
      const p = String(n).padStart(2, "0");
      setMm(p);
      if (hh.length === 2) emitComplete(hh, p);
    }
    if (e.key === "ArrowLeft" && (mRef.current && mRef.current.selectionStart === 0)) {
      e.preventDefault();
      hRef.current?.focus();
    }
  };

  return (
    <div className={cn("inline-flex items-center gap-1", className)}>
      <input
        ref={hRef}
        type="text"
        inputMode="numeric"
        pattern="[0-9]*"
        value={hh}
        onChange={onHChange}
        onBlur={onHBlur}
        onKeyDown={onHKeyDown}
        disabled={disabled}
        placeholder="HH"
        className={cn(
          "h-9 w-12 rounded-md border border-input bg-transparent text-sm shadow-sm px-2 text-center",
          "focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
        )}
        aria-label="Hours"
      />
      <span className="select-none text-muted-foreground">:</span>
      <input
        ref={mRef}
        type="text"
        inputMode="numeric"
        pattern="[0-9]*"
        value={mm}
        onChange={onMChange}
        onBlur={onMBlur}
        onKeyDown={onMKeyDown}
        disabled={disabled}
        placeholder="MM"
        className={cn(
          "h-9 w-12 rounded-md border border-input bg-transparent text-sm shadow-sm px-2 text-center",
          "focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
        )}
        aria-label="Minutes"
      />
    </div>
  );
}

export default TimeField;
