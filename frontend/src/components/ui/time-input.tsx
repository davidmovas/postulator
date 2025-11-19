"use client";

import React from "react";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Clock } from "lucide-react";
import { cn } from "@/lib/utils";

export interface TimeInputProps {
  value: string;
  onChange: (value: string) => void;
  isValid: boolean;
  className?: string;
}

export function validateAndCorrectTime(timeString: string): string {
  const cleaned = (timeString || "").replace(/[^\d:]/g, "");

  // Split by colon
  const parts = cleaned.split(":");

  if (parts.length !== 2) {
    return "00:00";
  }

  let hours = parseInt(parts[0], 10);
  let minutes = parseInt(parts[1], 10);

  // Handle invalid numbers
  if (isNaN(hours)) hours = 0;
  if (isNaN(minutes)) minutes = 0;

  // Clamp hours to 0-23
  if (hours > 23) hours = 23;
  if (hours < 0) hours = 0;

  // Clamp minutes to 0-59
  if (minutes > 59) minutes = 59;
  if (minutes < 0) minutes = 0;

  // Format to HH:MM
  return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}`;
}

export function isValidTimeFormat(timeString: string): boolean {
  const timeRegex = /^([01]?\d|2[0-3]):([0-5]?\d)$/;
  return timeRegex.test(timeString);
}

export function TimeInput({ value, onChange, isValid, className }: TimeInputProps) {
  const [internalValue, setInternalValue] = React.useState(value || "00:00");
  const [isFocused, setIsFocused] = React.useState(false);

  React.useEffect(() => {
    if (!value) {
      setInternalValue("00:00");
    } else if (value !== internalValue) {
      const corrected = forceFormat(value);
      setInternalValue(corrected);
    }
  }, [value]);

  const forceFormat = (raw: string): string => {
    const digits = (raw || "").replace(/\D/g, "").slice(0, 4);
    const h = digits.slice(0, 2);
    const m = digits.slice(2, 4);
    const hours = h.length ? parseInt(h, 10) : 0;
    const minutes = m.length ? parseInt(m, 10) : 0;
    const hh = String(Math.max(0, Math.min(23, isNaN(hours) ? 0 : hours))).padStart(2, "0");
    const mm = String(Math.max(0, Math.min(59, isNaN(minutes) ? 0 : minutes))).padStart(2, "0");
    return `${hh}:${mm}`;
  };

  const applyAndEmit = (raw: string) => {
    const correctedTime = forceFormat(raw);
    setInternalValue(correctedTime);
    onChange(correctedTime);
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const next = e.target.value;
    // Во время ввода не отправляем наружу каждое изменение, чтобы не мешать редактированию курсора
    // Обновляем часы/минуты на лету из цифр, сохраняя двоеточие
    const digits = (next || "").replace(/\D/g, "").slice(0, 4);
    if (digits.length === 0) {
      // Показываем 00:00 (сохраняем маску), но не эмитим наружу до blur
      setInternalValue("00:00");
      return;
    }
    let hh = "00";
    let mm = internalValue.split(":")[1] || "00";
    if (digits.length <= 2) {
      hh = String(parseInt(digits.padEnd(2, "0"), 10)).padStart(2, "0");
    } else {
      hh = digits.slice(0, 2);
      mm = digits.slice(2).padEnd(2, "0");
    }
    // Не жёстко клампим во время набора, но ограничим диапазоны, чтобы не вводить невозможные значения
    const hNum = Math.max(0, Math.min(23, parseInt(hh, 10) || 0));
    const mNum = Math.max(0, Math.min(59, parseInt(mm, 10) || 0));
    const view = `${String(hNum).padStart(2, "0")}:${String(mNum).padStart(2, "0")}`;
    setInternalValue(view);
  };

  const handleBlur = () => {
    applyAndEmit(internalValue);
  };

  const handleFocus = () => setIsFocused(true);
  const handleBlurWrapper = () => {
    setIsFocused(false);
    handleBlur();
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Разрешаем базовые клавиши навигации/редактирования
    if (
      e.key === "Backspace" ||
      e.key === "Delete" ||
      e.key === "Tab" ||
      e.key === "Escape" ||
      e.key === "Enter" ||
      e.key.startsWith("Arrow")
    ) {
      // На Enter применяем формат и эмитим наружу
      if (e.key === "Enter") {
        applyAndEmit(internalValue);
      }
      return;
    }
    // Разрешаем ввод только цифр (двоеточие добавляется автоматически)
    if (!/^\d$/.test(e.key)) {
      e.preventDefault();
    }
  };

  // Выбор по выпадающему селектору
  const setHours = (hours: number) => {
    const [, mm] = internalValue.split(":");
    const hh = String(Math.max(0, Math.min(23, hours))).padStart(2, "0");
    applyAndEmit(`${hh}:${mm || "00"}`);
  };
  const setMinutes = (minutes: number) => {
    const [hh] = internalValue.split(":");
    const mm = String(Math.max(0, Math.min(59, minutes))).padStart(2, "0");
    applyAndEmit(`${hh || "00"}:${mm}`);
  };

  const hours = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, "0"));
  const minutes = Array.from({ length: 12 }, (_, i) => String(i * 5).padStart(2, "0"));

  const [hhSelected, mmSelected] = internalValue.split(":");

  return (
    <div className="relative inline-flex items-center">
      <Input
        type="text"
        inputMode="numeric"
        value={internalValue}
        onChange={handleChange}
        onFocus={handleFocus}
        onBlur={handleBlurWrapper}
        onKeyDown={handleKeyDown}
        placeholder="00:00"
        maxLength={5}
        className={cn(
          "font-mono tabular-nums w-[120px] pr-9",
          !isValid && "border-destructive focus-visible:ring-destructive",
          className
        )}
        aria-label="Time input (24-hour format)"
        aria-invalid={!isValid}
      />

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            className="absolute right-0 mr-1.5 inline-flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            aria-label="Open time selector"
          >
            <Clock className="h-4 w-4" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="p-2 w-auto">
          <div className="grid grid-cols-2 gap-2">
            <div className="max-h-48 overflow-auto pr-1">
              <div className="px-2 py-1 text-xs text-muted-foreground">Hours</div>
              {hours.map((h) => (
                <DropdownMenuItem
                  key={h}
                  onSelect={() => setHours(parseInt(h))}
                  className={cn("font-mono tabular-nums", h === hhSelected && "bg-accent")}
                >
                  {h}
                </DropdownMenuItem>
              ))}
            </div>
            <div className="max-h-48 overflow-auto pl-1">
              <div className="px-2 py-1 text-xs text-muted-foreground">Minutes</div>
              {minutes.map((m) => (
                <DropdownMenuItem
                  key={m}
                  onSelect={() => setMinutes(parseInt(m))}
                  className={cn("font-mono tabular-nums", m === mmSelected && "bg-accent")}
                >
                  {m}
                </DropdownMenuItem>
              ))}
            </div>
          </div>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

export default TimeInput;
