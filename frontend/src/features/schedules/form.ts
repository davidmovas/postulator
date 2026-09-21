import type { Schedule } from "../../data/types.js";

export const cadenceKinds = ["interval", "cron"] as const;

export type CadenceKind = (typeof cadenceKinds)[number];

export const intervalUnits = ["minutes", "hours", "days"] as const;

export type IntervalUnit = (typeof intervalUnits)[number];

export const unitMinutes: Readonly<Record<IntervalUnit, number>> = { minutes: 1, hours: 60, days: 1440 };

export const maxTargets = 500;

export const defaultCron = "0 3 * * *";

export interface ScheduleDraft {
    name: string;
    cadence: CadenceKind;
    intervalValue: number;
    intervalUnit: IntervalUnit;
    cron: string;
    templateId: string;
    publishMode: string;
    status: string;
    entityId: string;
    limit: number;
    maxUsd: number;
    maxTokens: number;
    enabled: boolean;
}

export interface CreateFields {
    siteId: string;
    name: string;
    cron?: string;
    intervalMinutes?: number;
    entityId: string;
    status: string;
    limit: number;
    templateId: string;
    publishMode: string;
    maxUsd: number;
    maxTokens: number;
    enabled: boolean;
}

export interface UpdateFields {
    id: string;
    name?: string;
    cron?: string;
    intervalMinutes?: number;
    entityId?: string;
    status?: string;
    limit?: number;
    templateId?: string;
    publishMode?: string;
    maxUsd?: number;
    maxTokens?: number;
}

export function splitInterval(minutes: number): { value: number; unit: IntervalUnit } {
    if (minutes > 0 && minutes % unitMinutes.days === 0) {
        return { value: minutes / unitMinutes.days, unit: "days" };
    }
    if (minutes > 0 && minutes % unitMinutes.hours === 0) {
        return { value: minutes / unitMinutes.hours, unit: "hours" };
    }
    return { value: minutes > 0 ? minutes : 1, unit: minutes > 0 ? "minutes" : "days" };
}

export function joinInterval(value: number, unit: IntervalUnit): number {
    return Math.max(1, Math.round(value)) * unitMinutes[unit];
}

export function draftOf(schedule: Schedule | null): ScheduleDraft {
    if (schedule === null) {
        return {
            name: "",
            cadence: "interval",
            intervalValue: 1,
            intervalUnit: "days",
            cron: defaultCron,
            templateId: "",
            publishMode: "draft",
            status: "planned",
            entityId: "",
            limit: maxTargets,
            maxUsd: 0,
            maxTokens: 0,
            enabled: true,
        };
    }
    const cadence: CadenceKind = schedule.cron !== undefined && schedule.cron !== "" ? "cron" : "interval";
    const interval = splitInterval(schedule.intervalMinutes ?? 0);
    return {
        name: schedule.name,
        cadence,
        intervalValue: interval.value,
        intervalUnit: interval.unit,
        cron: schedule.cron === undefined || schedule.cron === "" ? defaultCron : schedule.cron,
        templateId: schedule.templateId ?? "",
        publishMode: schedule.publishMode === "" ? "draft" : schedule.publishMode,
        status: schedule.status ?? "",
        entityId: schedule.entityId ?? "",
        limit: schedule.limit > 0 ? schedule.limit : maxTargets,
        maxUsd: schedule.maxUsd,
        maxTokens: schedule.maxTokens,
        enabled: schedule.enabled,
    };
}

export function cadenceOf(draft: ScheduleDraft): { cron: string; intervalMinutes: number } {
    return draft.cadence === "cron"
        ? { cron: draft.cron.trim(), intervalMinutes: 0 }
        : { cron: "", intervalMinutes: joinInterval(draft.intervalValue, draft.intervalUnit) };
}

export function createOf(draft: ScheduleDraft, siteId: string): CreateFields {
    const cadence = cadenceOf(draft);
    const fields: CreateFields = {
        siteId,
        name: draft.name.trim(),
        entityId: draft.entityId,
        status: draft.status,
        limit: draft.limit,
        templateId: draft.templateId,
        publishMode: draft.publishMode,
        maxUsd: draft.maxUsd,
        maxTokens: draft.maxTokens,
        enabled: draft.enabled,
    };
    if (draft.cadence === "cron") {
        fields.cron = cadence.cron;
    } else {
        fields.intervalMinutes = cadence.intervalMinutes;
    }
    return fields;
}

export function updateOf(draft: ScheduleDraft, current: Schedule): UpdateFields {
    const fields: UpdateFields = { id: current.id };
    const cadence = cadenceOf(draft);
    const heldCron = current.cron ?? "";
    const heldInterval = current.intervalMinutes ?? 0;
    if (draft.cadence === "cron") {
        if (cadence.cron !== heldCron || heldInterval > 0) {
            fields.cron = cadence.cron;
        }
    } else if (cadence.intervalMinutes !== heldInterval || heldCron !== "") {
        fields.intervalMinutes = cadence.intervalMinutes;
    }
    if (draft.name.trim() !== current.name) {
        fields.name = draft.name.trim();
    }
    if (draft.entityId !== (current.entityId ?? "")) {
        fields.entityId = draft.entityId;
    }
    if (draft.status !== (current.status ?? "")) {
        fields.status = draft.status;
    }
    if (draft.limit !== current.limit) {
        fields.limit = draft.limit;
    }
    if (draft.templateId !== (current.templateId ?? "")) {
        fields.templateId = draft.templateId;
    }
    if (draft.publishMode !== current.publishMode) {
        fields.publishMode = draft.publishMode;
    }
    if (draft.maxUsd !== current.maxUsd) {
        fields.maxUsd = draft.maxUsd;
    }
    if (draft.maxTokens !== current.maxTokens) {
        fields.maxTokens = draft.maxTokens;
    }
    return fields;
}

export function dirty(draft: ScheduleDraft, current: Schedule | null): boolean {
    if (current === null) {
        return true;
    }
    return Object.keys(updateOf(draft, current)).length > 1;
}

export function ready(draft: ScheduleDraft): boolean {
    if (draft.name.trim() === "") {
        return false;
    }
    return draft.cadence === "cron" ? draft.cron.trim() !== "" : draft.intervalValue > 0;
}
