"use client";

import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { JobCreateInput, Schedule, OnceSchedule, IntervalSchedule, DailySchedule } from "@/models/jobs";
import { useState } from "react";

interface ScheduleSectionProps {
    formData: Partial<JobCreateInput>;
    onUpdate: (updates: Partial<JobCreateInput>) => void;
}

export function ScheduleSection({ formData, onUpdate }: ScheduleSectionProps) {
    const [scheduleType, setScheduleType] = useState<string>(formData.schedule?.type || "manual");

    const handleScheduleTypeChange = (type: string) => {
        setScheduleType(type);

        let newSchedule: Schedule | undefined;

        switch (type) {
            case "once":
                newSchedule = {
                    type: "once",
                    config: {
                        executeAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString() // Завтра
                    } as OnceSchedule
                };
                break;
            case "interval":
                newSchedule = {
                    type: "interval",
                    config: {
                        value: 1,
                        unit: "hours",
                        startAt: new Date().toISOString()
                    } as IntervalSchedule
                };
                break;
            case "daily":
                newSchedule = {
                    type: "daily",
                    config: {
                        hour: 9,
                        minute: 0,
                        weekdays: [1, 2, 3, 4, 5] // Пн-Пт
                    } as DailySchedule
                };
                break;
            default:
                newSchedule = undefined;
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
                                Execute at a specific date and time. The task will be triggered at the nearest specified time.
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

                {/* Once Schedule */}
                {scheduleType === "once" && formData.schedule?.config && (
                    <div className="space-y-4 border-t pt-4 pl-6">
                        <div className="space-y-2">
                            <Label htmlFor="executeAt">Execute At</Label>
                            <Input
                                id="executeAt"
                                type="datetime-local"
                                value={(formData.schedule.config as OnceSchedule).executeAt?.slice(0, 16)}
                                onChange={(e) => updateScheduleConfig({
                                    executeAt: new Date(e.target.value).toISOString()
                                })}
                            />
                        </div>
                    </div>
                )}

                {/* Interval Schedule */}
                {scheduleType === "interval" && formData.schedule?.config && (
                    <div className="space-y-4 border-t pt-4 pl-6">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                                <Label htmlFor="intervalValue">Interval</Label>
                                <Input
                                    id="intervalValue"
                                    type="number"
                                    min="1"
                                    value={(formData.schedule.config as IntervalSchedule).value || 1}
                                    onChange={(e) => updateScheduleConfig({
                                        value: parseInt(e.target.value)
                                    })}
                                />
                            </div>
                            <div className="space-y-2">
                                <Label htmlFor="intervalUnit">Unit</Label>
                                <Select
                                    value={(formData.schedule.config as IntervalSchedule).unit || "hours"}
                                    onValueChange={(value) => updateScheduleConfig({ unit: value })}
                                >
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="minutes">Minutes</SelectItem>
                                        <SelectItem value="hours">Hours</SelectItem>
                                        <SelectItem value="days">Days</SelectItem>
                                        <SelectItem value="weeks">Weeks</SelectItem>
                                        <SelectItem value="months">Months</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="startAt">Start At (Optional)</Label>
                            <Input
                                id="startAt"
                                type="datetime-local"
                                value={(formData.schedule.config as IntervalSchedule).startAt?.slice(0, 16) || ""}
                                onChange={(e) => updateScheduleConfig({
                                    startAt: e.target.value ? new Date(e.target.value).toISOString() : undefined
                                })}
                            />
                        </div>
                    </div>
                )}

                {/* Daily Schedule */}
                {scheduleType === "daily" && formData.schedule?.config && (
                    <div className="space-y-4 border-t pt-4 pl-6">
                        <div className="space-y-2">
                            <Label htmlFor="time">Time</Label>
                            <Input
                                id="time"
                                type="time"
                                step="60"
                                value={(() => {
                                    const cfg = formData.schedule!.config as DailySchedule;
                                    const h = (cfg.hour ?? 9).toString().padStart(2, '0');
                                    const m = (cfg.minute ?? 0).toString().padStart(2, '0');
                                    return `${h}:${m}`;
                                })()}
                                onChange={(e) => {
                                    const [h, m] = e.target.value.split(":").map(v => parseInt(v));
                                    updateScheduleConfig({ hour: h, minute: m });
                                }}
                            />
                        </div>
                        <div className="space-y-2">
                            <Label>Days of Week</Label>
                            <div className="flex flex-wrap gap-2">
                                {[
                                    { value: 1, label: 'Mon' },
                                    { value: 2, label: 'Tue' },
                                    { value: 3, label: 'Wed' },
                                    { value: 4, label: 'Thu' },
                                    { value: 5, label: 'Fri' },
                                    { value: 6, label: 'Sat' },
                                    { value: 0, label: 'Sun' }
                                ].map(day => {
                                    const current = (formData.schedule?.config as DailySchedule)?.weekdays || [];
                                    const selected = current.includes(day.value);
                                    return (
                                        <Button
                                            key={day.value}
                                            type="button"
                                            variant={selected ? "default" : "outline"}
                                            size="sm"
                                            className={selected ? "" : "text-muted-foreground"}
                                            onClick={() => {
                                                const newWeekdays = selected
                                                    ? current.filter(d => d !== day.value)
                                                    : [...current, day.value];
                                                updateScheduleConfig({ weekdays: newWeekdays });
                                            }}
                                        >
                                            {day.label}
                                        </Button>
                                    );
                                })}
                            </div>
                        </div>
                    </div>
                )}
            </CardContent>
        </Card>
    );
}