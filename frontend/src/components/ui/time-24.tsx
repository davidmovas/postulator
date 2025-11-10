"use client";

import { cn } from "@/lib/utils";
import { ClockIcon } from "lucide-react";
import { dateInputStyle } from "@/components/ui/datefield-rac";
import React from "react";

interface TimeInput24Props {
  value: string; // "HH:mm"
  onChange: (value: string) => void;
  className?: string;
  disabled?: boolean;
  invalid?: boolean;
}

// Clamp helper
function clamp(num: number, min: number, max: number) {
  return Math.max(min, Math.min(max, num));
}

// Normalize arbitrary string to HH:mm (24h). Returns valid string; falls back to 00:00
function normalizeHHmm(raw: string): string {
  if (!raw) return "00:00";
  const match = raw.match(/^(\d{1,2})(?::?(\d{1,2}))?/);
  if (!match) return "00:00";
  const h = clamp(parseInt(match[1] || "0", 10) || 0, 0, 23);
  const m = clamp(parseInt(match[2] || "0", 10) || 0, 0, 59);
  const hh = String(h).padStart(2, "0");
  const mm = String(m).padStart(2, "0");
  return `${hh}:${mm}`;
}

export function TimeInput24({ value, onChange, className, disabled, invalid }: TimeInput24Props) {
  // Internal text state to allow user typing without jumping
  const [text, setText] = React.useState<string>(value || "00:00");

  // Keep internal state in sync when parent value changes externally
  React.useEffect(() => {
    const normalized = normalizeHHmm(value || "00:00");
    setText(normalized);
  }, [value]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let val = e.target.value;
    // Remove any non-digits and colon, limit length
    val = val.replace(/[^0-9:]/g, "").slice(0, 5);
    // Auto-insert colon when typing 3rd char like 123 -> 12:3
    if (/^\d{3}$/.test(val.replace(":", ""))) {
      const d = val.replace(":", "");
      val = `${d.slice(0, 2)}:${d.slice(2)}`;
    }
    setText(val);

    // When value forms valid HH:mm, propagate normalized
    const parts = val.split(":");
    if (parts.length === 2 && parts[0].length === 2 && parts[1].length === 2) {
      const normalized = normalizeHHmm(val);
      if (normalized !== value) onChange(normalized);
    }
  };

  const handleBlur = () => {
    const normalized = normalizeHHmm(text);
    setText(normalized);
    if (normalized !== value) onChange(normalized);
  };

  return (
    <div className={cn("relative w-[120px]", className)}>
      <input
        type="text"
        inputMode="numeric"
        pattern="^([01]\\d|2[0-3]):[0-5]\\d$"
        placeholder="00:00"
        value={text}
        onChange={handleChange}
        onBlur={handleBlur}
        disabled={disabled}
        aria-invalid={invalid || undefined}
        className={cn(dateInputStyle, "ps-9 w-full font-mono", invalid && "border-destructive")}
      />
      <div className="pointer-events-none absolute inset-y-0 start-0 flex items-center justify-center ps-3 text-muted-foreground/80 data-[disabled=true]:opacity-50">
        <ClockIcon size={16} aria-hidden="true" />
      </div>
    </div>
  );
}

export default TimeInput24;
