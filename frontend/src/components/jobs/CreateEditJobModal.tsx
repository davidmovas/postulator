"use client";

import React, { useEffect, useMemo, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectTrigger,
    SelectValue,
    SelectContent,
    SelectItem,
} from "@/components/ui/select";
import { FieldSet, FieldGroup, FieldLegend, Field, FieldContent, FieldDescription } from "@/components/ui/field";
import { TimeField, TimeField24 } from "@/components/ui/time-field";
import { Job, createJob, updateJob } from "@/services/job";
import { ScheduleType, ScheduleTypeConst, JobStatusConst, JobStatus } from "@/constants/jobs";
import { useErrorHandling } from "@/lib/error-handling";
import { listSites, getSiteCategories, type Site, type Category } from "@/services/site";
import { listPrompts, type Prompt } from "@/services/prompt";
import { listActiveAIProviders, type AIProvider } from "@/services/aiProvider";

export interface CreateEditJobModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    job?: Job | null;
    onSaved?: () => void | Promise<void>;
}

// Типы для интервалов
type IntervalUnit = "hours" | "days" | "weeks" | "months";

export function CreateEditJobModal({ open, onOpenChange, job, onSaved }: CreateEditJobModalProps) {
    const isEdit = !!job;
    const { withErrorHandling } = useErrorHandling();

    // Form state
    const [name, setName] = useState("");
    const [nameUserModified, setNameUserModified] = useState(false);
    const [siteId, setSiteId] = useState<number>(0);
    const [previousSiteId, setPreviousSiteId] = useState<number>(0);
    const [categoryId, setCategoryId] = useState<number>(0);
    const [promptId, setPromptId] = useState<number>(0);
    const [aiProviderId, setAIProviderId] = useState<number>(0);
    const [aiModel, setAIModel] = useState("");
    const [requiresValidation, setRequiresValidation] = useState<boolean>(false);
    const [scheduleType, setScheduleType] = useState<ScheduleType>(ScheduleTypeConst.Interval);

    // Schedule state
    const [scheduleTime, setScheduleTime] = useState<string>("10:00");
    const [intervalValue, setIntervalValue] = useState<number>(1);
    const [intervalUnit, setIntervalUnit] = useState<IntervalUnit>("hours");
    const [weekdays, setWeekdays] = useState<number[]>([]);
    const [jitterEnabled, setJitterEnabled] = useState<boolean>(true);
    const [jitterMinutes, setJitterMinutes] = useState<number>(60);
    const [status, setStatus] = useState<JobStatus>(JobStatusConst.Active);

    // Options state
    const [sites, setSites] = useState<Site[]>([]);
    const [categories, setCategories] = useState<Category[]>([]);
    const [prompts, setPrompts] = useState<Prompt[]>([]);
    const [providers, setProviders] = useState<AIProvider[]>([]);

    const [loadingCategories, setLoadingCategories] = useState(false);
    const [saving, setSaving] = useState(false);

    // Load base lists when modal opens
    useEffect(() => {
        if (!open) return;
        (async () => {
            try {
                const [sitesRes, promptsRes, providersRes] = await Promise.all([
                    listSites(),
                    listPrompts(),
                    listActiveAIProviders(),
                ]);
                setSites(sitesRes);
                setPrompts(promptsRes);
                setProviders(providersRes);
            } catch {}
        })();
    }, [open]);

    // Initialize or reset form on open/job change
    useEffect(() => {
        if (!open) return;
        if (job) {
            setName(job.name || "");
            setNameUserModified(true);
            setSiteId(job.siteId || 0);
            setPreviousSiteId(job.siteId || 0);
            setCategoryId(job.categoryId || 0);
            setPromptId(job.promptId || 0);
            setAIProviderId(job.aiProviderId || 0);
            setAIModel(job.aiModel || "");
            setRequiresValidation(job.requiresValidation);
            setScheduleType(job.scheduleType || ScheduleTypeConst.Interval);

            // Schedule initialization
            const hh = job.scheduleHour ?? 10;
            const mm = job.scheduleMinute ?? 0;
            setScheduleTime(`${String(hh).padStart(2,"0")}:${String(mm).padStart(2,"0")}`);

            setIntervalValue(job.intervalValue ?? 1);
            setIntervalUnit((job.intervalUnit as IntervalUnit) || "hours");
            setWeekdays(job.weekdays || []);
            setJitterEnabled(job.jitterEnabled ?? true);
            setJitterMinutes(job.jitterMinutes || 60);
            setStatus((job.status as JobStatus) || JobStatusConst.Active);
        } else {
            setName("");
            setNameUserModified(false);
            setSiteId(0);
            setPreviousSiteId(0);
            setCategoryId(0);
            setPromptId(0);
            setAIProviderId(0);
            setAIModel("");
            setRequiresValidation(false);
            setScheduleType(ScheduleTypeConst.Interval);
            setScheduleTime("10:00");
            setIntervalValue(1);
            setIntervalUnit("hours");
            setWeekdays([]);
            setJitterEnabled(true);
            setJitterMinutes(60);
            setStatus(JobStatusConst.Active);
        }
    }, [open, job]);

    // Auto-fill name from site
    useEffect(() => {
        if (!open || nameUserModified) return;

        const selectedSite = sites.find(s => s.id === siteId);
        if (selectedSite && siteId !== previousSiteId) {
            setName(selectedSite.name);
            setPreviousSiteId(siteId);
        }
    }, [open, siteId, sites, nameUserModified, previousSiteId]);

    // Handle manual name changes
    const handleNameChange = (value: string) => {
        setName(value);
        if (value.trim().length > 0) {
            setNameUserModified(true);
        }
    };

    // Load categories when site changes
    useEffect(() => {
        if (!open) return;
        if (siteId > 0) {
            setLoadingCategories(true);
            (async () => {
                try {
                    const cats = await getSiteCategories(siteId);
                    setCategories(cats);
                    if (!cats.some((c) => c.id === categoryId)) {
                        setCategoryId(cats[0]?.id || 0);
                    }
                } catch {
                    setCategories([]);
                    setCategoryId(0);
                } finally {
                    setLoadingCategories(false);
                }
            })();
        } else {
            setCategories([]);
            setCategoryId(0);
        }
    }, [open, siteId]);

    // When provider changes, auto-adopt its configured model
    useEffect(() => {
        if (!open) return;
        const provider = providers.find((p) => p.id === aiProviderId);
        if (provider) {
            setAIModel(provider.model || "");
        } else {
            setAIModel("");
        }
    }, [open, aiProviderId, providers]);

    const isValidTime = (v?: string) => {
        if (!v) return false;
        const m = v.match(/^\s*(\d{1,2}):(\d{2})\s*$/);
        if (!m) return false;
        const hh = parseInt(m[1], 10);
        const mm = parseInt(m[2], 10);
        return hh >= 0 && hh <= 23 && mm >= 0 && mm <= 59;
    };

    const canSave = useMemo(() => {
        const baseOk = (
            name.trim().length > 0 &&
            siteId > 0 &&
            categories.length > 0 &&
            categoryId > 0 &&
            prompts.length > 0 &&
            promptId > 0 &&
            providers.length > 0 &&
            aiProviderId > 0 &&
            aiModel.trim().length > 0
        );

        // Валидация для разных типов расписания
        let scheduleOk = true;

        switch (scheduleType) {
            case ScheduleTypeConst.Once:
                scheduleOk = isValidTime(scheduleTime);
                break;
            case ScheduleTypeConst.Interval:
                scheduleOk = intervalValue >= 1 && !!intervalUnit;
                break;
            case ScheduleTypeConst.Daily:
                scheduleOk = weekdays.length > 0 && isValidTime(scheduleTime);
                break;
            case ScheduleTypeConst.Manual:
                scheduleOk = true;
                break;
        }

        return baseOk && scheduleOk;
    }, [
        name, siteId, categories.length, categoryId, prompts.length, promptId,
        providers.length, aiProviderId, aiModel, scheduleType, scheduleTime,
        intervalValue, intervalUnit, weekdays.length
    ]);

    const normalizePayload = () => {
        const jm = Math.max(0, Math.min(180, jitterMinutes || 0));

        let scheduleHour: number | undefined = undefined;
        let scheduleMinute: number | undefined = undefined;

        // Parse time for schedules that need it
        if ((scheduleType === ScheduleTypeConst.Once || scheduleType === ScheduleTypeConst.Daily) && isValidTime(scheduleTime)) {
            const [h, m] = scheduleTime.split(":");
            scheduleHour = parseInt(h, 10);
            scheduleMinute = parseInt(m, 10);
        }

        const base: any = {
            name: name.trim(),
            siteId,
            categoryId,
            promptId,
            aiProviderId,
            aiModel: aiModel.trim(),
            requiresValidation,
            scheduleType: scheduleType,
            jitterEnabled,
            jitterMinutes: jm,
            status: status,
        };

        // Очистка полей в зависимости от типа расписания
        switch (scheduleType) {
            case ScheduleTypeConst.Manual:
                return {
                    ...base,
                    intervalValue: undefined,
                    intervalUnit: undefined,
                    scheduleHour: undefined,
                    scheduleMinute: undefined,
                    weekdays: undefined,
                };

            case ScheduleTypeConst.Once:
                return {
                    ...base,
                    scheduleHour,
                    scheduleMinute,
                    intervalValue: undefined,
                    intervalUnit: undefined,
                    weekdays: undefined,
                };

            case ScheduleTypeConst.Interval:
                return {
                    ...base,
                    intervalValue: Math.max(1, intervalValue),
                    intervalUnit: intervalUnit,
                    scheduleHour: undefined,
                    scheduleMinute: undefined,
                    weekdays: undefined,
                };

            case ScheduleTypeConst.Daily:
                return {
                    ...base,
                    scheduleHour,
                    scheduleMinute,
                    weekdays: weekdays.length > 0 ? weekdays : [1, 2, 3, 4, 5], // default to weekdays
                    intervalValue: undefined,
                    intervalUnit: undefined,
                };

            default:
                return base;
        }
    };

    const handleSave = async () => {
        if (!canSave) return;
        setSaving(true);
        try {
            await withErrorHandling(async () => {
                const payload = normalizePayload();

                if (isEdit && job) {
                    await updateJob({
                        id: job.id,
                        ...payload,
                        createdAt: job.createdAt,
                        updatedAt: new Date().toISOString(),
                        lastRunAt: job.lastRunAt,
                        nextRunAt: job.nextRunAt,
                    });
                } else {
                    await createJob(payload);
                }

                if (onSaved) await onSaved();
                onOpenChange(false);
            }, {
                successMessage: isEdit ? "Job updated" : "Job created",
                showSuccess: true
            });
        } finally {
            setSaving(false);
        }
    };

    const noSites = sites.length === 0;

    // Обработчик изменения дней недели
    const handleWeekdayChange = (dayValue: number, checked: boolean) => {
        if (checked) {
            setWeekdays(prev => [...prev, dayValue].sort());
        } else {
            setWeekdays(prev => prev.filter(d => d !== dayValue));
        }
    };

    return (
        <Dialog open={open} onOpenChange={(o) => !saving && onOpenChange(o)}>
            <DialogContent className="w-[50vw] max-w-7xl max-h-[90vh] overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>{isEdit ? "Edit Job" : "Create Job"}</DialogTitle>
                    <DialogDescription>
                        {isEdit ? "Update job configuration and schedule." : "Fill in details to create a new job."}
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-6">
                    <FieldSet>
                        <FieldLegend>General</FieldLegend>
                        <FieldGroup>
                            <Field>
                                <FieldContent>
                                    <Label>Name</Label>
                                    <Input
                                        value={name}
                                        onChange={(e) => handleNameChange(e.target.value)}
                                        placeholder="Job name (auto-filled from site)"
                                    />
                                    <FieldDescription>
                                        Auto-filled from site name, or enter custom name
                                    </FieldDescription>
                                </FieldContent>
                            </Field>
                        </FieldGroup>
                    </FieldSet>

                    <FieldSet>
                        <FieldLegend>Targets</FieldLegend>
                        <FieldGroup className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <Field>
                                <FieldContent>
                                    <Label>Site</Label>
                                    <Select
                                        value={siteId ? String(siteId) : undefined}
                                        onValueChange={(v) => setSiteId(parseInt(v, 10))}
                                    >
                                        <SelectTrigger>
                                            <SelectValue placeholder={noSites ? "No sites available" : "Select site"} />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {sites.map((s) => (
                                                <SelectItem key={s.id} value={String(s.id)}>{s.name}</SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    {noSites && (
                                        <FieldDescription>You need to add a Site before creating a Job.</FieldDescription>
                                    )}
                                </FieldContent>
                            </Field>

                            <Field>
                                <FieldContent>
                                    <Label>Category</Label>
                                    <Select
                                        value={categoryId ? String(categoryId) : undefined}
                                        onValueChange={(v) => setCategoryId(parseInt(v, 10))}
                                        disabled={siteId === 0 || loadingCategories || categories.length === 0}
                                    >
                                        <SelectTrigger>
                                            <SelectValue placeholder={
                                                siteId === 0 ? "Select site first" :
                                                    loadingCategories ? "Loading..." :
                                                        categories.length ? "Select category" : "No categories"
                                            } />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {categories.map((c) => (
                                                <SelectItem key={c.id} value={String(c.id)}>{c.name}</SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                </FieldContent>
                            </Field>

                            <Field>
                                <FieldContent>
                                    <Label>Prompt</Label>
                                    <Select
                                        value={promptId ? String(promptId) : undefined}
                                        onValueChange={(v) => setPromptId(parseInt(v, 10))}
                                        disabled={prompts.length === 0}
                                    >
                                        <SelectTrigger>
                                            <SelectValue placeholder={prompts.length === 0 ? "No prompts available" : "Select prompt"} />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {prompts.map((p) => (
                                                <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                </FieldContent>
                            </Field>
                        </FieldGroup>
                    </FieldSet>

                    <FieldSet>
                        <FieldLegend>AI</FieldLegend>
                        <FieldGroup className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <Field>
                                <FieldContent>
                                    <Label>Provider</Label>
                                    <Select
                                        value={aiProviderId ? String(aiProviderId) : undefined}
                                        onValueChange={(v) => setAIProviderId(parseInt(v, 10))}
                                        disabled={providers.length === 0}
                                    >
                                        <SelectTrigger>
                                            <SelectValue placeholder={providers.length === 0 ? "No providers" : "Select provider"} />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {providers.map((p) => (
                                                <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                </FieldContent>
                            </Field>

                            <Field>
                                <FieldContent>
                                    <Label>Model</Label>
                                    <Input
                                        value={aiModel}
                                        onChange={(e) => setAIModel(e.target.value)}
                                        placeholder="Enter AI model name"
                                        disabled
                                    />
                                </FieldContent>
                            </Field>

                            <Field>
                                <FieldContent>
                                    <Label>Status</Label>
                                    <Select value={status} onValueChange={(v) => setStatus(v as JobStatus)}>
                                        <SelectTrigger>
                                            <SelectValue placeholder="Select status" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value={JobStatusConst.Active}>Active</SelectItem>
                                            <SelectItem value={JobStatusConst.Paused}>Paused</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </FieldContent>
                            </Field>
                        </FieldGroup>
                    </FieldSet>

                    <FieldSet>
                        <FieldLegend>Schedule</FieldLegend>
                        <FieldGroup>
                            <Field>
                                <FieldContent>
                                    <Label>Schedule Type</Label>
                                    <Select value={scheduleType} onValueChange={(v) => setScheduleType(v as ScheduleType)}>
                                        <SelectTrigger>
                                            <SelectValue placeholder="Select schedule type" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value={ScheduleTypeConst.Interval}>Interval</SelectItem>
                                            <SelectItem value={ScheduleTypeConst.Manual}>Manual</SelectItem>
                                            <SelectItem value={ScheduleTypeConst.Once}>Once</SelectItem>
                                            <SelectItem value={ScheduleTypeConst.Daily}>Daily</SelectItem>
                                        </SelectContent>
                                    </Select>
                                    <FieldDescription>
                                        {scheduleType === ScheduleTypeConst.Interval && "Job will run automatically every specified interval"}
                                        {scheduleType === ScheduleTypeConst.Manual && "Job will only run when manually triggered"}
                                        {scheduleType === ScheduleTypeConst.Once && "Job will run once at the specified time"}
                                        {scheduleType === ScheduleTypeConst.Daily && "Job will run on selected days at specified time"}
                                    </FieldDescription>
                                </FieldContent>
                            </Field>

                            {scheduleType === ScheduleTypeConst.Interval && (
                                <Field>
                                    <FieldContent>
                                        <Label>Run Every</Label>
                                        <div className="grid grid-cols-2 gap-2">
                                            <Input
                                                type="number"
                                                min={1}
                                                value={intervalValue}
                                                onChange={(e) => {
                                                    const val = parseInt(e.target.value, 10);
                                                    setIntervalValue(isNaN(val) ? 1 : Math.max(1, val));
                                                }}
                                                placeholder="Value"
                                            />
                                            <Select
                                                value={intervalUnit}
                                                onValueChange={(v) => setIntervalUnit(v as IntervalUnit)}
                                            >
                                                <SelectTrigger>
                                                    <SelectValue placeholder="Unit" />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    <SelectItem value="hours">Hours</SelectItem>
                                                    <SelectItem value="days">Days</SelectItem>
                                                    <SelectItem value="weeks">Weeks</SelectItem>
                                                    <SelectItem value="months">Months</SelectItem>
                                                </SelectContent>
                                            </Select>
                                        </div>
                                        <FieldDescription>
                                            How often to run this job automatically
                                        </FieldDescription>
                                    </FieldContent>
                                </Field>
                            )}

                            {(scheduleType === ScheduleTypeConst.Once || scheduleType === ScheduleTypeConst.Daily) && (
                                <Field>
                                    <FieldContent>
                                        <Label>Run At</Label>
                                        <TimeField24
                                            value={scheduleTime}
                                            onChange={setScheduleTime}
                                            disabled={false}
                                        />
                                        <FieldDescription>
                                            {scheduleType === ScheduleTypeConst.Once
                                                ? "Job will run once at this time"
                                                : "Job will run at this time on selected days"}
                                        </FieldDescription>
                                    </FieldContent>
                                </Field>
                            )}

                            {scheduleType === ScheduleTypeConst.Daily && (
                                <Field>
                                    <FieldContent>
                                        <Label>On Days</Label>
                                        <div className="flex flex-wrap gap-4">
                                            {[
                                                { value: 1, label: 'Mon' },
                                                { value: 2, label: 'Tue' },
                                                { value: 3, label: 'Wed' },
                                                { value: 4, label: 'Thu' },
                                                { value: 5, label: 'Fri' },
                                                { value: 6, label: 'Sat' },
                                                { value: 7, label: 'Sun' }
                                            ].map(day => (
                                                <div key={day.value} className="flex items-center gap-2">
                                                    <Checkbox
                                                        id={`weekday-${day.value}`}
                                                        checked={weekdays.includes(day.value)}
                                                        onCheckedChange={(checked) =>
                                                            handleWeekdayChange(day.value, checked === true)
                                                        }
                                                    />
                                                    <Label
                                                        htmlFor={`weekday-${day.value}`}
                                                        className="text-sm cursor-pointer min-w-[40px]"
                                                    >
                                                        {day.label}
                                                    </Label>
                                                </div>
                                            ))}
                                        </div>
                                        <FieldDescription>
                                            Select at least one day of the week
                                        </FieldDescription>
                                    </FieldContent>
                                </Field>
                            )}
                        </FieldGroup>
                    </FieldSet>

                    <FieldSet>
                        <FieldLegend>Options</FieldLegend>
                        <FieldGroup className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <Field>
                                <FieldContent>
                                    <Label>Validation</Label>
                                    <div className="flex items-center gap-2 pt-2">
                                        <Checkbox
                                            id="requiresValidation"
                                            checked={requiresValidation}
                                            onCheckedChange={(v) => setRequiresValidation(!!v)}
                                        />
                                        <Label htmlFor="requiresValidation" className="text-sm cursor-pointer">
                                            Require manual approval before publishing
                                        </Label>
                                    </div>
                                </FieldContent>
                            </Field>
                            <Field>
                                <FieldContent>
                                    <Label>Jitter</Label>
                                    <div className="space-y-2">
                                        <div className="flex items-center gap-2">
                                            <Checkbox
                                                id="jitter"
                                                checked={jitterEnabled}
                                                onCheckedChange={(v) => setJitterEnabled(!!v)}
                                            />
                                            <Label htmlFor="jitter" className="text-sm cursor-pointer">
                                                Add random delay to execution time
                                            </Label>
                                        </div>
                                        {jitterEnabled && (
                                            <div className="flex items-center gap-2">
                                                <span className="text-sm text-muted-foreground whitespace-nowrap">Up to</span>
                                                <Input
                                                    type="number"
                                                    min={0}
                                                    max={180}
                                                    value={jitterMinutes}
                                                    onChange={(e) => {
                                                        const val = parseInt(e.target.value, 10);
                                                        setJitterMinutes(isNaN(val) ? 0 : Math.max(0, Math.min(180, val)));
                                                    }}
                                                    className="w-20"
                                                />
                                                <span className="text-sm text-muted-foreground whitespace-nowrap">minutes</span>
                                            </div>
                                        )}
                                    </div>
                                </FieldContent>
                            </Field>
                        </FieldGroup>
                    </FieldSet>

                    <div className="flex justify-end gap-2 pt-4 border-t">
                        <Button
                            variant="outline"
                            onClick={() => onOpenChange(false)}
                            disabled={saving}
                        >
                            Cancel
                        </Button>
                        <Button
                            onClick={handleSave}
                            disabled={!canSave || saving || noSites}
                        >
                            {saving ? "Saving..." : (isEdit ? "Save Changes" : "Create Job")}
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}

export default CreateEditJobModal;