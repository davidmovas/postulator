"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar-rac";
import { DateInput, dateInputStyle } from "@/components/ui/datefield-rac";
import { Button as AriaButton, DatePicker, Dialog, Group, Popover, I18nProvider } from "react-aria-components";
import { CalendarDate } from "@internationalized/date";
import { JobCreateInput, Schedule, OnceSchedule, IntervalSchedule, DailySchedule } from "@/models/jobs";
import { useEffect, useMemo, useState } from "react";
import { CalendarIcon } from "lucide-react";
import TimeInput from "@/components/ui/time-input";

interface ScheduleSectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
}

export function ScheduleSection({ formData, onUpdate }: ScheduleSectionProps) {
    const DateTimePicker = ({
        label,
        date,
        time,
        onDateChange,
        onTimeChange,
        requireFuture = false,
        invalidHint,
    }: {
        label: string;
        date: Date | undefined;
        time: string;
        onDateChange: (d: Date) => void;
        onTimeChange: (t: string) => void;
        requireFuture?: boolean;
        invalidHint?: string;
    }) => {
        // Convert local Date to CalendarDate using local year/month/day to avoid timezone shifts
        const dateToCalendarDate = (d: Date): CalendarDate => new CalendarDate(d.getFullYear(), d.getMonth() + 1, d.getDate());
        const calendarDateToDate = (cd: CalendarDate): Date => new Date(cd.year, cd.month - 1, cd.day);
        const ariaValue = date ? dateToCalendarDate(date) : undefined;

        const iso = date ? combineDateTimeToISO(date, time) : undefined;
        const isInvalid = requireFuture && !!date && !!iso && new Date(iso).getTime() <= Date.now();

        return (
            <div className="space-y-2">
                <Label className="text-sm font-medium">{label}</Label>
                <div className="flex flex-col sm:flex-row sm:items-center gap-3">
                    <I18nProvider locale="en-GB">
                        <DatePicker
                            value={ariaValue as any}
                            onChange={(val: any) => {
                                if (!val) return;
                                const d = calendarDateToDate(val as CalendarDate);
                                onDateChange(d);
                            }}
                        >
                            <div className="flex">
                                <Group className={dateInputStyle + " pe-9 w-[200px]"}>
                                    <DateInput unstyled />
                                </Group>
                                <AriaButton aria-label="Open calendar" className="z-10 -ms-9 -me-px flex w-9 items-center justify-center rounded-e-md text-muted-foreground/80 transition-[color,box-shadow] outline-none hover:text-foreground data-focus-visible:border-ring data-focus-visible:ring-[3px] data-focus-visible:ring-ring/50">
                                    <CalendarIcon size={16} />
                                </AriaButton>
                            </div>
                            <Popover
                                className="z-50 rounded-md border bg-background text-popover-foreground shadow-lg outline-hidden data-entering:animate-in data-exiting:animate-out data-[entering]:fade-in-0 data-[entering]:zoom-in-95 data-[exiting]:fade-out-0 data-[exiting]:zoom-out-95"
                                offset={4}
                            >
                                <Dialog className="max-h-[inherit] overflow-auto p-2">
                                    <Calendar />
                                </Dialog>
                            </Popover>
                        </DatePicker>
                    </I18nProvider>

                    <TimeInput value={time} onChange={onTimeChange} isValid={!isInvalid} />
                </div>
                {isInvalid && (
                    <p className="text-xs text-destructive">{invalidHint || "Date and time must be in the future."}</p>
                )}
            </div>
        );
    };

    const WeekdayPills = ({
        value,
        onChange,
    }: { value: number[]; onChange: (days: number[]) => void }) => {
        const days = [
            { value: 0, label: "Mon" },
            { value: 1, label: "Tue" },
            { value: 2, label: "Wed" },
            { value: 3, label: "Thu" },
            { value: 4, label: "Fri" },
            { value: 5, label: "Sat" },
            { value: 6, label: "Sun" },
        ];
        return (
            <div className="flex flex-wrap gap-2">
                {days.map((day) => {
                    const selected = value.includes(day.value);
                    return (
                        <Button
                            key={day.value}
                            type="button"
                            variant={selected ? "default" : "outline"}
                            size="sm"
                            className={"rounded-full px-3 " + (selected ? "" : "text-muted-foreground")}
                            onClick={() => {
                                const next = selected ? value.filter((d) => d !== day.value) : [...value, day.value];
                                onChange(next);
                            }}
                        >
                            {day.label}
                        </Button>
                    );
                })}
            </div>
        );
    };
    const [scheduleType, setScheduleType] = useState<string>(formData.schedule?.type || "manual");

    // Ensure schedule object exists so it gets sent to backend
    useEffect(() => {
        if (!formData.schedule) {
            onUpdate({ schedule: { type: "manual", config: {} } as Schedule });
        }
    }, [formData.schedule]);

    // Helpers to parse/format local date + time to ISO and back
    const toTimeString = (date: Date | undefined) => {
        if (!date) return "12:00";
        const h = date.getHours().toString().padStart(2, '0');
        const m = date.getMinutes().toString().padStart(2, '0');
        return `${h}:${m}`;
    };

    // Build RFC3339 string with LOCAL timezone offset, e.g. 2025-11-20T03:34:00+01:00 (not Z)
    const toLocalRFC3339 = (d: Date): string => {
        const pad = (n: number) => String(n).padStart(2, "0");
        const year = d.getFullYear();
        const month = pad(d.getMonth() + 1);
        const day = pad(d.getDate());
        const hour = pad(d.getHours());
        const minute = pad(d.getMinutes());
        const second = pad(d.getSeconds());
        const ms = String(d.getMilliseconds()).padStart(3, "0");
        const offMinTotal = -d.getTimezoneOffset(); // minutes east of UTC
        const sign = offMinTotal >= 0 ? "+" : "-";
        const abs = Math.abs(offMinTotal);
        const offH = pad(Math.floor(abs / 60));
        const offM = pad(abs % 60);
        return `${year}-${month}-${day}T${hour}:${minute}:${second}.${ms}${sign}${offH}:${offM}`;
    };

    const combineDateTimeToISO = (date: Date | undefined, timeHHMM: string): string | undefined => {
        if (!date || !timeHHMM) return undefined;
        const [hh, mm] = timeHHMM.split(":").map((v) => parseInt(v));
        const combined = new Date(date);
        combined.setHours(hh || 0, mm || 0, 0, 0);
        // return LOCAL time with offset instead of UTC Z
        return toLocalRFC3339(combined);
    };

    const parseISOToDate = (iso?: string): Date | undefined => {
        if (!iso) return undefined;
        const d = new Date(iso);
        return isNaN(d.getTime()) ? undefined : d;
    };

    // Derived state for Once
    const onceDate = useMemo(() => {
        const cfg = formData.schedule?.config as any;
        const val: string | undefined = cfg?.executeAt || cfg?.execute_at;
        return parseISOToDate(val);
    }, [formData.schedule]);
    const onceTime = useMemo(() => toTimeString(onceDate), [onceDate]);

    const intervalStartDate = useMemo(() => {
        const cfg = formData.schedule?.config as any;
        const val: string | undefined = cfg?.startAt || cfg?.start_at;
        return parseISOToDate(val);
    }, [formData.schedule]);
    const intervalStartTime = useMemo(() => {
        // Если стартовая дата не задана, показываем текущее время + 10 минут как дефолт
        const base = intervalStartDate || new Date(Date.now() + 10 * 60 * 1000);
        return toTimeString(base);
    }, [intervalStartDate]);

    const isOnceInPast = useMemo(() => {
        const iso = combineDateTimeToISO(onceDate, onceTime);
        if (!iso) return false;
        return new Date(iso).getTime() <= Date.now();
    }, [onceDate, onceTime]);

    const handleScheduleTypeChange = (type: string) => {
        setScheduleType(type);

        let newSchedule: Schedule | undefined;

        switch (type) {
            case "manual":
                newSchedule = {
                    type: "manual",
                    config: {}
                } as Schedule;
                break;
            case "once":
                newSchedule = {
                    type: "once",
                    config: {
                        // По умолчанию: завтра, текущее время + 10 минут (локальное RFC3339 с оффсетом)
                        executeAt: (() => toLocalRFC3339(new Date(Date.now() + 24 * 60 * 60 * 1000 + 10 * 60 * 1000)))(),
                        // Для совместимости с сервером (snake_case)
                        execute_at: (() => toLocalRFC3339(new Date(Date.now() + 24 * 60 * 60 * 1000 + 10 * 60 * 1000)))(),
                    } as OnceSchedule
                };
                break;
            case "interval":
                newSchedule = {
                    type: "interval",
                    config: {
                        value: 1,
                        unit: "hours",
                        // Старт по умолчанию: текущее время + 10 минут (локальное RFC3339)
                        startAt: (() => toLocalRFC3339(new Date(Date.now() + 10 * 60 * 1000)))(),
                        // Для совместимости с сервером (snake_case)
                        start_at: (() => toLocalRFC3339(new Date(Date.now() + 10 * 60 * 1000)))(),
                    } as IntervalSchedule
                };
                break;
            case "daily":
                newSchedule = {
                    type: "daily",
                    config: {
                        hour: 9,
                        minute: 0,
                        weekdays: [0, 1, 2, 3, 4] // Mon-Fri (0-based, Mon=0)
                    } as DailySchedule
                };
                break;
            default:
                newSchedule = { type: "manual", config: {} } as Schedule;
        }

        onUpdate({ schedule: newSchedule });
    };

    const updateScheduleConfig = (updates: any) => {
        if (!formData.schedule) return;

        onUpdate({
            schedule: {
                ...formData.schedule,
                config: {
                    ...formData.schedule.config,
                    ...updates
                }
            }
        });
    };

    return (
        <Card>
            <CardHeader>
                <CardTitle>Schedule</CardTitle>
                <CardFooter>
                    Configure when and how often the job should run
                </CardFooter>
            </CardHeader>
            <CardContent className="space-y-6">
                <RadioGroup
                    value={scheduleType}
                    onValueChange={handleScheduleTypeChange}
                    className="space-y-3"
                >
                    <div className="flex items-center space-x-2">
                        <RadioGroupItem value="manual" id="manual" />
                        <Label htmlFor="manual" className="flex-1">
                            <div className="font-medium">Manual Only</div>
                            <div className="text-sm text-muted-foreground">
                                Run only when manually triggered
                            </div>
                        </Label>
                    </div>

                    <div className="flex items-center space-x-2">
                        <RadioGroupItem value="once" id="once" />
                        <Label htmlFor="once" className="flex-1">
                            <div className="font-medium">Run Once</div>
                            <div className="text-sm text-muted-foreground">
                                Execute at a specific date and time
                            </div>
                        </Label>
                    </div>

                    <div className="flex items-center space-x-2">
                        <RadioGroupItem value="interval" id="interval" />
                        <Label htmlFor="interval" className="flex-1">
                            <div className="font-medium">Interval</div>
                            <div className="text-sm text-muted-foreground">
                                Run repeatedly at regular intervals
                            </div>
                        </Label>
                    </div>

                    <div className="flex items-center space-x-2">
                        <RadioGroupItem value="daily" id="daily" />
                        <Label htmlFor="daily" className="flex-1">
                            <div className="font-medium">Daily</div>
                            <div className="text-sm text-muted-foreground">
                                Run at specific times on selected days
                            </div>
                        </Label>
                    </div>
                </RadioGroup>

                {/* Once Schedule - React Aria styled (single date) */}
                {scheduleType === "once" && formData.schedule?.config && (
                    <div className="space-y-4 border-t pt-4 pl-6">
                        <DateTimePicker
                            label="Execute at"
                            date={onceDate}
                            time={onceTime}
                            requireFuture
                            invalidHint="Please pick a future date and time."
                            onDateChange={(d) => {
                                const iso = combineDateTimeToISO(d, onceTime);
                                if (iso) updateScheduleConfig({ executeAt: iso, execute_at: iso });
                            }}
                            onTimeChange={(t) => {
                                const iso = combineDateTimeToISO(onceDate || new Date(), t);
                                if (iso) updateScheduleConfig({ executeAt: iso, execute_at: iso });
                            }}
                        />
                    </div>
                )}

                {/* Interval Schedule */}
                {scheduleType === "interval" && formData.schedule?.config && (
                    <div className="space-y-4 border-t pt-4 pl-6">
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
                            <div className="space-y-2 w-full sm:w-auto">
                                <Label htmlFor="intervalValue">Interval</Label>
                                <Input
                                    id="intervalValue"
                                    type="number"
                                    min="1"
                                    value={Math.max(1, Number((formData.schedule.config as IntervalSchedule).value || 1))}
                                    onChange={(e) => updateScheduleConfig({
                                        value: Math.max(1, parseInt(e.target.value || "1"))
                                    })}
                                    className="w-[120px]"
                                />
                            </div>
                            <div className="space-y-2 w-full sm:w-auto">
                                <Label htmlFor="intervalUnit">Unit</Label>
                                <Select
                                    value={(formData.schedule.config as IntervalSchedule).unit || "hours"}
                                    onValueChange={(value) => updateScheduleConfig({ unit: value })}
                                >
                                    <SelectTrigger className="w-[160px]">
                                        <SelectValue placeholder="Select unit" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="hours">Hours</SelectItem>
                                        <SelectItem value="days">Days</SelectItem>
                                        <SelectItem value="weeks">Weeks</SelectItem>
                                        <SelectItem value="months">Months</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>
                            <div className="space-y-2 grow">
                                <Label>Start at</Label>
                                <DateTimePicker
                                    label=""
                                    date={intervalStartDate}
                                    time={intervalStartTime}
                                    requireFuture={!!intervalStartDate}
                                    invalidHint="Start time must be in the future."
                                    onDateChange={(d) => {
                                        const iso = combineDateTimeToISO(d, intervalStartTime);
                                        updateScheduleConfig({ startAt: iso, start_at: iso });
                                    }}
                                    onTimeChange={(t) => {
                                        const iso = combineDateTimeToISO(intervalStartDate || new Date(), t);
                                        updateScheduleConfig({ startAt: iso, start_at: iso });
                                    }}
                                />
                            </div>
                        </div>
                    </div>
                )}

                {/* Daily Schedule */}
                {scheduleType === "daily" && formData.schedule?.config && (
                    <div className="space-y-4 border-t pt-4 pl-6">
                        <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
                            <div className="space-y-2 w-full sm:w-auto">
                                <Label htmlFor="time">Time</Label>
                                {(() => {
                                    const cfg = formData.schedule!.config as DailySchedule;
                                    const h = (cfg.hour ?? 9).toString().padStart(2, '0');
                                    const m = (cfg.minute ?? 0).toString().padStart(2, '0');
                                    const value = `${h}:${m}`;
                                    return (
                                        <TimeInput
                                            value={value}
                                            onChange={(val) => {
                                                const [hh, mm] = val.split(":").map((x) => parseInt(x));
                                                updateScheduleConfig({ hour: hh, minute: mm });
                                            }}
                                            isValid={true}
                                        />
                                    );
                                })()}
                            </div>
                            <div className="space-y-2 w-full sm:w-auto">
                                <Label>Days of week</Label>
                                {(() => {
                                    const current = (formData.schedule?.config as DailySchedule)?.weekdays || [];
                                    const none = current.length === 0;
                                    return (
                                        <div className="space-y-2">
                                            <WeekdayPills
                                                value={current}
                                                onChange={(days) => updateScheduleConfig({ weekdays: days })}
                                            />
                                            {none && (
                                                <p className="text-xs text-destructive">Select at least one day.</p>
                                            )}
                                        </div>
                                    );
                                })()}
                            </div>
                        </div>
                    </div>
                )}
            </CardContent>
        </Card>
    );
}