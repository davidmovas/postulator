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
import { TimeField } from "@/components/ui/time-field";
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

// Helpers to normalize time formats
function toHHMM(value?: string | null): string {
  if (!value) return "";
  const parts = String(value).split(":");
  const hh = (parts[0] || "00").padStart(2, "0");
  const mm = (parts[1] || "00").padStart(2, "0");
  return `${hh}:${mm}`;
}

function toHHMMSS(value?: string | null): string | undefined {
  if (!value) return undefined;
  const parts = String(value).split(":");
  const hh = (parts[0] || "00").padStart(2, "0");
  const mm = (parts[1] || "00").padStart(2, "0");
  return `${hh}:${mm}:00`;
}

export function CreateEditJobModal({ open, onOpenChange, job, onSaved }: CreateEditJobModalProps) {
  const isEdit = !!job;
  const { withErrorHandling } = useErrorHandling();

  // Form state
  const [name, setName] = useState("");
  const [siteId, setSiteId] = useState<number>(0);
  const [categoryId, setCategoryId] = useState<number>(0);
  const [promptId, setPromptId] = useState<number>(0);
  const [aiProviderId, setAIProviderId] = useState<number>(0);
  const [aiModel, setAIModel] = useState("");
  const [requiresValidation, setRequiresValidation] = useState<boolean>(false);
  const [scheduleType, setScheduleType] = useState<ScheduleType>(ScheduleTypeConst.Manual);
  const [scheduleTime, setScheduleTime] = useState<string>(""); // used for "once" HH:MM
  const [intervalValue, setIntervalValue] = useState<number | undefined>(undefined);
  const [intervalUnit, setIntervalUnit] = useState<string | undefined>(undefined);
  const [jitterEnabled, setJitterEnabled] = useState<boolean>(false);
  const [jitterMinutes, setJitterMinutes] = useState<number>(0);
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
      setSiteId(job.siteId || 0);
      setCategoryId(job.categoryId || 0);
      setPromptId(job.promptId || 0);
      setAIProviderId(job.aiProviderId || 0);
      setAIModel(job.aiModel || "");
      setRequiresValidation(!!job.requiresValidation);
      setScheduleType(job.scheduleType || ScheduleTypeConst.Manual);
      const hh = job.scheduleHour ?? undefined;
      const mm = job.scheduleMinute ?? undefined;
      setScheduleTime(hh !== undefined && mm !== undefined ? `${String(hh).padStart(2,"0")}:${String(mm).padStart(2,"0")}` : "");
      setIntervalValue(job.intervalValue ?? undefined);
      setIntervalUnit(job.intervalUnit ?? undefined);
      setJitterEnabled(!!job.jitterEnabled);
      setJitterMinutes(job.jitterMinutes || 30);
      setStatus((job.status as JobStatus) || JobStatusConst.Active);
    } else {
      setName("");
      setSiteId(0);
      setCategoryId(0);
      setPromptId(0);
      setAIProviderId(0);
      setAIModel("");
      setRequiresValidation(false);
      setScheduleType(ScheduleTypeConst.Manual);
      setScheduleTime("10:00"); // used if/when switching to Once
      setIntervalValue(undefined);
      setIntervalUnit(undefined);
      setJitterEnabled(false);
      setJitterMinutes(30);
      setStatus(JobStatusConst.Active);
    }
  }, [open, job]);

  // Load categories when site changes
  useEffect(() => {
    if (!open) return;
    if (siteId > 0) {
      setLoadingCategories(true);
      (async () => {
        try {
          const cats = await getSiteCategories(siteId);
          setCategories(cats);
          // Reset category if it doesn't belong
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

  const needsTime = useMemo(() => scheduleType === ScheduleTypeConst.Once, [scheduleType]);

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
    const timeOk = scheduleType === ScheduleTypeConst.Once ? isValidTime(scheduleTime) : true;
    const intervalOk = scheduleType === ScheduleTypeConst.Interval ? !!intervalValue && (intervalValue as number) >= 1 && !!intervalUnit : true;
    return baseOk && timeOk && intervalOk;
  }, [name, siteId, categories.length, categoryId, prompts.length, promptId, providers.length, aiProviderId, aiModel, scheduleType, scheduleTime, intervalValue, intervalUnit]);

  const normalizePayload = () => {
    const jm = Math.max(0, Math.min(180, jitterMinutes || 0));

    // derive hour/minute from scheduleTime if present
    let scheduleHour: number | undefined = undefined;
    let scheduleMinute: number | undefined = undefined;
    if (scheduleType === ScheduleTypeConst.Once && isValidTime(scheduleTime)) {
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
      status: status || "active",
    };

    if (scheduleType === ScheduleTypeConst.Manual) {
      return {
        ...base,
        intervalValue: undefined,
        intervalUnit: undefined,
        scheduleHour: undefined,
        scheduleMinute: undefined,
      };
    }

    if (scheduleType === ScheduleTypeConst.Once) {
      return {
        ...base,
        scheduleHour,
        scheduleMinute,
        intervalValue: undefined,
        intervalUnit: undefined,
      };
    }

    // Interval
    const val = Math.max(1, parseInt(String(intervalValue || 0), 10));
    const unit = (intervalUnit || "minutes").toLowerCase();
    return {
      ...base,
      intervalValue: val,
      intervalUnit: unit,
      scheduleHour: undefined,
      scheduleMinute: undefined,
    };
  };

  const handleSave = async () => {
    if (!canSave) return;
    setSaving(true);
    try {
      await withErrorHandling(async () => {
        if (isEdit && job) {
          await updateJob({
            id: job.id,
            ...normalizePayload(),
            createdAt: job.createdAt,
            updatedAt: job.updatedAt,
            lastRunAt: job.lastRunAt,
            nextRunAt: job.nextRunAt,
          } as any);
        } else {
          await createJob({
            ...normalizePayload(),
          } as any);
        }
        if (onSaved) await onSaved();
        onOpenChange(false);
      }, { successMessage: isEdit ? "Job updated" : "Job created", showSuccess: true });
    } finally {
      setSaving(false);
    }
  };

  const noSites = sites.length === 0;


  return (
    <Dialog open={open} onOpenChange={(o) => !saving && onOpenChange(o)}>
      <DialogContent className="max-w-4xl">
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
                  <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="My job" />
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
                  <Select value={siteId ? String(siteId) : undefined} onValueChange={(v) => setSiteId(parseInt(v, 10))}>
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
                      <SelectValue placeholder={siteId === 0 ? "Select site first" : (loadingCategories ? "Loading..." : (categories.length ? "Select category" : "No categories"))} />
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
                  <Input value={aiModel} readOnly placeholder="Select provider to see model" />
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
                      <SelectItem value={JobStatusConst.Active}>active</SelectItem>
                      <SelectItem value={JobStatusConst.Paused}>paused</SelectItem>
                    </SelectContent>
                  </Select>
                </FieldContent>
              </Field>
            </FieldGroup>
          </FieldSet>

          <FieldSet>
            <FieldLegend>Schedule</FieldLegend>
            <FieldGroup className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Field>
                <FieldContent>
                  <Label>Type</Label>
                  <Select value={scheduleType} onValueChange={(v) => setScheduleType(v as ScheduleType)}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select type" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={ScheduleTypeConst.Manual}>Manual</SelectItem>
                      <SelectItem value={ScheduleTypeConst.Once}>Once</SelectItem>
                      <SelectItem value={ScheduleTypeConst.Interval}>Interval</SelectItem>
                    </SelectContent>
                  </Select>
                </FieldContent>
              </Field>
              {scheduleType === ScheduleTypeConst.Once && (
                <Field>
                  <FieldContent>
                    <Label>Time</Label>
                    <TimeField
                      value={scheduleTime}
                      onChange={(v) => setScheduleTime(v)}
                      disabled={false}
                    />
                  </FieldContent>
                </Field>
              )}

              {scheduleType === ScheduleTypeConst.Interval && (
                <Field>
                  <FieldContent>
                    <Label>Interval</Label>
                    <div className="grid grid-cols-2 gap-2 items-center">
                      <Input
                        type="number"
                        min={1}
                        value={intervalValue ?? ''}
                        onChange={(e) => {
                          const n = Math.max(1, parseInt(e.target.value || '1', 10));
                          setIntervalValue(Number.isFinite(n) ? n : 1);
                        }}
                        placeholder="Value"
                      />
                      <Select
                        value={intervalUnit}
                        onValueChange={(v) => setIntervalUnit(v)}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Unit" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="minutes">minutes</SelectItem>
                          <SelectItem value="hours">hours</SelectItem>
                          <SelectItem value="days">days</SelectItem>
                          <SelectItem value="weeks">weeks</SelectItem>
                          <SelectItem value="months">months</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </FieldContent>
                </Field>
              )}
            </FieldGroup>
          </FieldSet>

          <FieldSet>
            <FieldLegend>Validation & Jitter</FieldLegend>
            <FieldGroup className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Field>
                <FieldContent>
                  <Label>Requires Validation</Label>
                  <div className="flex items-center gap-2 pt-2">
                    <Checkbox id="requiresValidation" checked={requiresValidation} onCheckedChange={(v) => setRequiresValidation(!!v)} />
                    <label htmlFor="requiresValidation" className="text-sm">Manual approval before publish</label>
                  </div>
                </FieldContent>
              </Field>
              <Field>
                <FieldContent>
                  <Label>Jitter Minutes</Label>
                  <div className="grid grid-cols-2 gap-2 items-center">
                    <div className="flex items-center gap-2">
                      <Checkbox id="jitter" checked={jitterEnabled} onCheckedChange={(v) => setJitterEnabled(!!v)} />
                      <label htmlFor="jitter" className="text-sm">Enable</label>
                    </div>
                    <Input
                      type="number"
                      min={0}
                      max={180}
                      value={jitterMinutes}
                      disabled={!jitterEnabled}
                      onChange={(e) => {
                        const v = e.target.value;
                        const n = Math.max(0, Math.min(180, parseInt(v || "0", 10)));
                        setJitterMinutes(n);
                      }}
                    />
                  </div>
                </FieldContent>
              </Field>
            </FieldGroup>
          </FieldSet>

          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={saving}>Cancel</Button>
            <Button onClick={handleSave} disabled={!canSave || saving || noSites}>{isEdit ? "Save Changes" : "Create"}</Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default CreateEditJobModal;
