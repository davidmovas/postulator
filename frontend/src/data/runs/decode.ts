import type { EventPayloads } from "../../generated/events.js";

export const runEventTypes = [
    "run.queued",
    "run.started",
    "run.paused",
    "run.resumed",
    "run.cancelled",
    "run.completed",
    "run.failed",
    "run.budget_exceeded",
    "item.started",
    "item.done",
    "item.failed",
    "item.needs_human",
    "step.started",
    "step.done",
    "step.failed",
    "step.retrying",
    "llm.usage",
] as const;

export type RunEventType = (typeof runEventTypes)[number];

export interface RunEventRecord {
    seq: number;
    type: RunEventType;
    at: string;
    payload: unknown;
}

export interface RawRunEvent {
    seq?: unknown;
    type?: unknown;
    at?: unknown;
    payload?: unknown;
}

const known: ReadonlySet<string> = new Set<string>(runEventTypes);

export const terminalRunEventTypes: ReadonlySet<RunEventType> = new Set<RunEventType>([
    "run.completed",
    "run.failed",
    "run.cancelled",
]);

export function seqOf(row: RawRunEvent): number | null {
    const held = row.seq;
    return typeof held === "number" && Number.isFinite(held) && held > 0 ? held : null;
}

export function decode(row: RawRunEvent): RunEventRecord | null {
    const seq = seqOf(row);
    if (seq === null) {
        return null;
    }
    const type = row.type;
    if (typeof type !== "string" || !known.has(type)) {
        return null;
    }
    const at = typeof row.at === "string" ? row.at : "";
    return { seq, type: type as RunEventType, at, payload: row.payload };
}

export function payloadOf<T extends RunEventType>(record: RunEventRecord, type: T): EventPayloads[T] | null {
    if (record.type !== type) {
        return null;
    }
    if (typeof record.payload !== "object" || record.payload === null) {
        return null;
    }
    return record.payload as EventPayloads[T];
}

