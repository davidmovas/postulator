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

interface TimeField24Props {
    value?: string;
    onChange?: (value: string) => void;
    disabled?: boolean;
    className?: string;
    placeholder?: string;
}

function isValidHour(hour: number): boolean {
    return hour >= 0 && hour <= 23;
}

function isValidMinute(minute: number): boolean {
    return minute >= 0 && minute <= 59;
}

export function TimeField24({
                                value = "",
                                onChange,
                                disabled,
                                className,
                                placeholder = "00:00"
                            }: TimeField24Props) {
    const { hh: initH, mm: initM } = useMemo(() => parseHHMM(value), [value]);
    const [hh, setHh] = useState(initH);
    const [mm, setMm] = useState(initM);

    const hRef = useRef<HTMLInputElement | null>(null);
    const mRef = useRef<HTMLInputElement | null>(null);

    // Синхронизация внутреннего состояния при изменении внешнего значения
    useEffect(() => {
        const { hh: h, mm: m } = parseHHMM(value);
        setHh(h);
        setMm(m);
    }, [value]);

    // Эмитим значение только когда время полное (обе части заполнены) или полностью очищено
    const emitComplete = (h: string, m: string) => {
        if (!onChange) return;

        if (h === "" && m === "") {
            onChange("");
            return;
        }

        const hOk = /^\d{1,2}$/.test(h);
        const mOk = /^\d{1,2}$/.test(m);

        if (!hOk || !mOk) return; // не эмитим частичные значения

        const hhNum = clamp(parseInt(h, 10), 0, 23);
        const mmNum = clamp(parseInt(m, 10), 0, 59);

        const formattedTime = `${String(hhNum).padStart(2, "0")}:${String(mmNum).padStart(2, "0")}`;
        onChange(formattedTime);
    };

    const onHChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const v = e.target.value.replace(/[^0-9]/g, "");

        if (v.length <= 2) {
            setHh(v);

            // Автопереход к минутам при вводе двух цифр
            if (v.length === 2) {
                mRef.current?.focus();
            }

            // Эмитим если обе части заполнены
            if (v.length === 2 && mm.length === 2) {
                emitComplete(v, mm);
            }
        }
    };

    const onMChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const v = e.target.value.replace(/[^0-9]/g, "");

        if (v.length <= 2) {
            setMm(v);

            // Эмитим если обе части заполнены
            if (v.length === 2 && hh.length === 2) {
                emitComplete(hh, v);
            }
        }
    };

    const onHBlur = () => {
        if (hh === "" && mm === "") {
            if (onChange) onChange("");
            return;
        }

        if (hh !== "") {
            const hourNum = clamp(parseInt(hh || "0", 10) || 0, 0, 23);
            const formattedHour = String(hourNum).padStart(2, "0");
            setHh(formattedHour);

            if (mm.length === 2) {
                emitComplete(formattedHour, mm);
            }
        }
    };

    const onMBlur = () => {
        if (hh === "" && mm === "") {
            if (onChange) onChange("");
            return;
        }

        if (mm !== "") {
            const minuteNum = clamp(parseInt(mm || "0", 10) || 0, 0, 59);
            const formattedMinute = String(minuteNum).padStart(2, "0");
            setMm(formattedMinute);

            if (hh.length === 2) {
                emitComplete(hh, formattedMinute);
            }
        }
    };

    const onHKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        // Обработка двоеточия для перехода к минутам
        if (e.key === ":" || e.key === "Enter") {
            e.preventDefault();
            mRef.current?.focus();
            return;
        }

        // Стрелки вверх/вниз для изменения часов
        if (e.key === "ArrowUp" || e.key === "ArrowDown") {
            e.preventDefault();
            let currentHour = parseInt(hh || "0", 10) || 0;
            currentHour = e.key === "ArrowUp" ? currentHour + 1 : currentHour - 1;

            // Циклическое изменение 0-23
            if (currentHour > 23) currentHour = 0;
            if (currentHour < 0) currentHour = 23;

            const formattedHour = String(currentHour).padStart(2, "0");
            setHh(formattedHour);

            if (mm.length === 2) {
                emitComplete(formattedHour, mm);
            }
        }

        // Стрелка вправо для перехода к минутам
        if (e.key === "ArrowRight" && (hh.length === 2 || (hRef.current && hRef.current.selectionStart === 2))) {
            e.preventDefault();
            mRef.current?.focus();
        }

        // Backspace в пустом поле часов - фокус остается
        if (e.key === "Backspace" && hh.length === 0) {
            e.preventDefault();
        }
    };

    const onMKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        // Обработка двоеточия
        if (e.key === ":") {
            e.preventDefault();
            return;
        }

        // Enter для подтверждения
        if (e.key === "Enter") {
            e.preventDefault();
            onMBlur();
            return;
        }

        // Стрелки вверх/вниз для изменения минут
        if (e.key === "ArrowUp" || e.key === "ArrowDown") {
            e.preventDefault();
            let currentMinute = parseInt(mm || "0", 10) || 0;
            currentMinute = e.key === "ArrowUp" ? currentMinute + 1 : currentMinute - 1;

            // Циклическое изменение 0-59
            if (currentMinute > 59) currentMinute = 0;
            if (currentMinute < 0) currentMinute = 59;

            const formattedMinute = String(currentMinute).padStart(2, "0");
            setMm(formattedMinute);

            if (hh.length === 2) {
                emitComplete(hh, formattedMinute);
            }
        }

        // Стрелка влево для перехода к часам
        if (e.key === "ArrowLeft" && (mRef.current && mRef.current.selectionStart === 0)) {
            e.preventDefault();
            hRef.current?.focus();
        }

        // Backspace в пустом поле минут - переход к часам
        if (e.key === "Backspace" && mm.length === 0) {
            e.preventDefault();
            hRef.current?.focus();
        }
    };

    // Показываем плейсхолдер, когда оба поля пустые
    const showPlaceholder = hh === "" && mm === "";

    return (
        <div className={cn("relative inline-flex items-center", className)}>
            {showPlaceholder && (
                <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                    <span className="text-muted-foreground text-sm">{placeholder}</span>
                </div>
            )}

            <div className={cn(
                "inline-flex items-center gap-1 border border-input rounded-md bg-background px-3 py-2",
                "focus-within:outline-none focus-within:ring-2 focus-within:ring-ring focus-within:border-ring",
                disabled && "opacity-50 cursor-not-allowed"
            )}>
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
                    className={cn(
                        "w-8 bg-transparent text-center text-sm outline-none border-none p-0",
                        "focus:outline-none focus:ring-0",
                        showPlaceholder && "text-transparent"
                    )}
                    aria-label="Часы (00-23)"
                    maxLength={2}
                />

                <span className="text-muted-foreground select-none">:</span>

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
                    className={cn(
                        "w-8 bg-transparent text-center text-sm outline-none border-none p-0",
                        "focus:outline-none focus:ring-0",
                        showPlaceholder && "text-transparent"
                    )}
                    aria-label="Минуты (00-59)"
                    maxLength={2}
                />
            </div>
        </div>
    );
}

export function TimeSelect24({
                                 value = "",
                                 onChange,
                                 disabled,
                                 className
                             }: TimeField24Props) {
    const [hour, setHour] = useState("00");
    const [minute, setMinute] = useState("00");

    // Генерация опций для часов и минут
    const hourOptions = Array.from({ length: 24 }, (_, i) =>
        String(i).padStart(2, "0")
    );

    const minuteOptions = Array.from({ length: 60 }, (_, i) =>
        String(i).padStart(2, "0")
    );

    useEffect(() => {
        if (value) {
            const [h, m] = value.split(":");
            if (h && m) {
                setHour(h);
                setMinute(m);
            }
        }
    }, [value]);

    const handleHourChange = (h: string) => {
        setHour(h);
        if (onChange) {
            onChange(`${h}:${minute}`);
        }
    };

    const handleMinuteChange = (m: string) => {
        setMinute(m);
        if (onChange) {
            onChange(`${hour}:${m}`);
        }
    };

    return (
        <div className={cn("flex items-center gap-2", className)}>
            <select
                value={hour}
                onChange={(e) => handleHourChange(e.target.value)}
                disabled={disabled}
                className="h-9 rounded-md border border-input bg-background px-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            >
                {hourOptions.map((h) => (
                    <option key={h} value={h}>
                        {h}
                    </option>
                ))}
            </select>

            <span className="text-muted-foreground">:</span>

            <select
                value={minute}
                onChange={(e) => handleMinuteChange(e.target.value)}
                disabled={disabled}
                className="h-9 rounded-md border border-input bg-background px-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            >
                {minuteOptions.map((m) => (
                    <option key={m} value={m}>
                        {m}
                    </option>
                ))}
            </select>
        </div>
    );
}

export default TimeField;
