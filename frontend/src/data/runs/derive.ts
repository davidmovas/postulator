import type { RunEventRecord } from "./decode.js";
import { payloadOf } from "./decode.js";

export type ItemState = "pending" | "running" | "retrying" | "needs_human" | "done" | "failed";

export interface StepFailure {
    code: string;
    message: string;
}

export interface ItemProgress {
    itemId: string;
    step: string | null;
    stepStartedAt: string | null;
    attempt: number;
    lastError: StepFailure | null;
    state: ItemState;
}

export interface LiveStats {
    items: number;
    done: number;
    failed: number;
    needsHuman: number;
    tokens: number;
    usd: number;
    calls: number;
}

export type RunPhase =
    | "queued"
    | "running"
    | "paused"
    | "cancelled"
    | "completed"
    | "failed"
    | "budget_exceeded";

type Events = readonly RunEventRecord[];

const progressCache = new WeakMap<Events, ReadonlyMap<string, ItemProgress>>();
const statsCache = new WeakMap<Events, LiveStats>();
const phaseCache = new WeakMap<Events, RunPhase | null>();

function blank(itemId: string): ItemProgress {
    return { itemId, step: null, stepStartedAt: null, attempt: 0, lastError: null, state: "pending" };
}

function reach(into: Map<string, ItemProgress>, itemId: string): ItemProgress {
    const held = into.get(itemId);
    if (held !== undefined) {
        return held;
    }
    const created = blank(itemId);
    into.set(itemId, created);
    return created;
}

export function itemProgress(events: Events): ReadonlyMap<string, ItemProgress> {
    const cached = progressCache.get(events);
    if (cached !== undefined) {
        return cached;
    }
    const byItem = new Map<string, ItemProgress>();
    for (const record of events) {
        switch (record.type) {
            case "item.started": {
                const payload = payloadOf(record, "item.started");
                if (payload !== null) {
                    const held = reach(byItem, payload.itemId);
                    held.state = "running";
                }
                break;
            }
            case "item.done": {
                const payload = payloadOf(record, "item.done");
                if (payload !== null) {
                    reach(byItem, payload.itemId).state = "done";
                }
                break;
            }
            case "item.failed": {
                const payload = payloadOf(record, "item.failed");
                if (payload !== null) {
                    const held = reach(byItem, payload.itemId);
                    held.state = "failed";
                    held.lastError = { code: payload.code, message: payload.message };
                }
                break;
            }
            case "item.needs_human": {
                const payload = payloadOf(record, "item.needs_human");
                if (payload !== null) {
                    reach(byItem, payload.itemId).state = "needs_human";
                }
                break;
            }
            case "item.restarted": {
                const payload = payloadOf(record, "item.restarted");
                if (payload !== null) {
                    byItem.set(payload.itemId, blank(payload.itemId));
                }
                break;
            }
            case "step.started": {
                const payload = payloadOf(record, "step.started");
                if (payload !== null) {
                    const held = reach(byItem, payload.itemId);
                    held.step = payload.step;
                    held.stepStartedAt = record.at;
                    held.state = "running";
                }
                break;
            }
            case "step.done": {
                const payload = payloadOf(record, "step.done");
                if (payload !== null) {
                    const held = reach(byItem, payload.itemId);
                    held.step = payload.step;
                }
                break;
            }
            case "step.retrying": {
                const payload = payloadOf(record, "step.retrying");
                if (payload !== null) {
                    const held = reach(byItem, payload.itemId);
                    held.step = payload.step;
                    held.attempt = payload.attempt;
                    held.state = "retrying";
                }
                break;
            }
            case "step.failed": {
                const payload = payloadOf(record, "step.failed");
                if (payload !== null) {
                    const held = reach(byItem, payload.itemId);
                    held.step = payload.step;
                    held.lastError = { code: payload.code, message: payload.message };
                }
                break;
            }
            default:
                break;
        }
    }
    const frozen: ReadonlyMap<string, ItemProgress> = byItem;
    progressCache.set(events, frozen);
    return frozen;
}

export function liveStats(events: Events): LiveStats {
    const cached = statsCache.get(events);
    if (cached !== undefined) {
        return cached;
    }
    const totals: LiveStats = { items: 0, done: 0, failed: 0, needsHuman: 0, tokens: 0, usd: 0, calls: 0 };
    for (const record of events) {
        switch (record.type) {
            case "run.queued": {
                const payload = payloadOf(record, "run.queued");
                if (payload !== null) {
                    totals.items = payload.items;
                }
                break;
            }
            case "item.done":
                totals.done += 1;
                break;
            case "item.failed":
                totals.failed += 1;
                break;
            case "item.needs_human":
                totals.needsHuman += 1;
                break;
            default:
                break;
        }
    }
    const frozen = Object.freeze(totals);
    statsCache.set(events, frozen);
    return frozen;
}

export function runPhase(events: Events): RunPhase | null {
    const cached = phaseCache.get(events);
    if (cached !== undefined) {
        return cached;
    }
    let phase: RunPhase | null = null;
    for (const record of events) {
        switch (record.type) {
            case "run.queued":
                phase = "queued";
                break;
            case "run.started":
            case "run.resumed":
                phase = "running";
                break;
            case "run.paused":
                phase = "paused";
                break;
            case "run.budget_exceeded":
                phase = "budget_exceeded";
                break;
            case "run.cancelled":
                phase = "cancelled";
                break;
            case "run.completed":
                phase = "completed";
                break;
            case "run.failed":
                phase = "failed";
                break;
            default:
                break;
        }
    }
    phaseCache.set(events, phase);
    return phase;
}
