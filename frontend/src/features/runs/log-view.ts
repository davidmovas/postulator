import type { RunEventRecord, RunEventType } from "../../data/runs/decode.js";
import { payloadOf } from "../../data/runs/decode.js";

type Events = readonly RunEventRecord[];

export interface RetryNotice {
    step: string;
    attempt: number;
    afterMs: number;
    at: string;
}

export function retryNotices(events: Events): ReadonlyMap<string, RetryNotice> {
    const byItem = new Map<string, RetryNotice>();
    for (const record of events) {
        if (record.type === "step.retrying") {
            const payload = payloadOf(record, "step.retrying");
            if (payload !== null) {
                byItem.set(payload.itemId, {
                    step: payload.step,
                    attempt: payload.attempt,
                    afterMs: payload.afterMs,
                    at: record.at,
                });
            }
            continue;
        }
        if (record.type === "step.started") {
            const payload = payloadOf(record, "step.started");
            if (payload !== null) {
                byItem.delete(payload.itemId);
            }
        }
    }
    return byItem;
}

export interface StepEntry {
    step: string;
    startedAt: string;
    finishedAt: string | null;
    durationMs: number | null;
    attempts: number;
    code: string | null;
    message: string | null;
}

export function stepTimeline(events: Events, itemId: string): readonly StepEntry[] {
    const out: StepEntry[] = [];
    let open: StepEntry | null = null;
    for (const record of events) {
        switch (record.type) {
            case "step.started": {
                const payload = payloadOf(record, "step.started");
                if (payload === null || payload.itemId !== itemId) {
                    break;
                }
                if (open !== null && open.step === payload.step) {
                    open.attempts += 1;
                    break;
                }
                open = {
                    step: payload.step,
                    startedAt: record.at,
                    finishedAt: null,
                    durationMs: null,
                    attempts: 1,
                    code: null,
                    message: null,
                };
                out.push(open);
                break;
            }
            case "step.done": {
                const payload = payloadOf(record, "step.done");
                if (payload === null || payload.itemId !== itemId || open === null) {
                    break;
                }
                open.finishedAt = record.at;
                open.durationMs = payload.durationMs;
                open = null;
                break;
            }
            case "step.failed": {
                const payload = payloadOf(record, "step.failed");
                if (payload === null || payload.itemId !== itemId || open === null) {
                    break;
                }
                open.finishedAt = record.at;
                open.code = payload.code;
                open.message = payload.message;
                open = null;
                break;
            }
            default:
                break;
        }
    }
    return out;
}

export interface FeedEntry {
    seq: number;
    type: RunEventType;
    at: string;
    itemId: string | null;
    step: string | null;
    code: string | null;
    message: string | null;
    reason: string | null;
    durationMs: number | null;
    attempt: number | null;
    afterMs: number | null;
    usd: number | null;
    items: number | null;
}

function blank(record: RunEventRecord): FeedEntry {
    return {
        seq: record.seq,
        type: record.type,
        at: record.at,
        itemId: null,
        step: null,
        code: null,
        message: null,
        reason: null,
        durationMs: null,
        attempt: null,
        afterMs: null,
        usd: null,
        items: null,
    };
}

export function describe(record: RunEventRecord): FeedEntry {
    const entry = blank(record);
    switch (record.type) {
        case "run.queued": {
            const payload = payloadOf(record, "run.queued");
            if (payload !== null) {
                entry.items = payload.items;
            }
            return entry;
        }
        case "run.paused": {
            const payload = payloadOf(record, "run.paused");
            if (payload !== null) {
                entry.reason = payload.reason;
            }
            return entry;
        }
        case "run.failed": {
            const payload = payloadOf(record, "run.failed");
            if (payload !== null) {
                entry.code = payload.code;
                entry.message = payload.message;
            }
            return entry;
        }
        case "run.completed": {
            const payload = payloadOf(record, "run.completed");
            if (payload !== null) {
                entry.items = payload.succeeded + payload.failed;
            }
            return entry;
        }
        case "run.budget_exceeded": {
            const payload = payloadOf(record, "run.budget_exceeded");
            if (payload !== null) {
                entry.usd = payload.spentUsd;
            }
            return entry;
        }
        case "item.started": {
            const payload = payloadOf(record, "item.started");
            if (payload !== null) {
                entry.itemId = payload.itemId;
            }
            return entry;
        }
        case "item.done": {
            const payload = payloadOf(record, "item.done");
            if (payload !== null) {
                entry.itemId = payload.itemId;
            }
            return entry;
        }
        case "item.failed": {
            const payload = payloadOf(record, "item.failed");
            if (payload !== null) {
                entry.itemId = payload.itemId;
                entry.code = payload.code;
                entry.message = payload.message;
            }
            return entry;
        }
        case "item.needs_human": {
            const payload = payloadOf(record, "item.needs_human");
            if (payload !== null) {
                entry.itemId = payload.itemId;
                entry.reason = payload.reason;
            }
            return entry;
        }
        case "step.started": {
            const payload = payloadOf(record, "step.started");
            if (payload !== null) {
                entry.itemId = payload.itemId;
                entry.step = payload.step;
            }
            return entry;
        }
        case "step.done": {
            const payload = payloadOf(record, "step.done");
            if (payload !== null) {
                entry.itemId = payload.itemId;
                entry.step = payload.step;
                entry.durationMs = payload.durationMs;
            }
            return entry;
        }
        case "step.failed": {
            const payload = payloadOf(record, "step.failed");
            if (payload !== null) {
                entry.itemId = payload.itemId;
                entry.step = payload.step;
                entry.code = payload.code;
                entry.message = payload.message;
            }
            return entry;
        }
        case "step.retrying": {
            const payload = payloadOf(record, "step.retrying");
            if (payload !== null) {
                entry.itemId = payload.itemId;
                entry.step = payload.step;
                entry.attempt = payload.attempt;
                entry.afterMs = payload.afterMs;
            }
            return entry;
        }
        default:
            return entry;
    }
}

export function feed(events: Events, limit: number, itemId: string | null = null): readonly FeedEntry[] {
    const out: FeedEntry[] = [];
    for (let index = events.length - 1; index >= 0 && out.length < limit; index -= 1) {
        const entry = describe(events[index]);
        if (itemId !== null && entry.itemId !== itemId) {
            continue;
        }
        out.push(entry);
    }
    return out;
}
