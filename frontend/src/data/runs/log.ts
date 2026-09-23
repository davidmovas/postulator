import type { TransportError } from "../../lib/errors.js";
import { isCancellation } from "../call.js";
import { listRunEvents } from "../endpoints/runs.js";
import { failure } from "../errors.js";
import type { RawRunEvent, RunEventRecord } from "./decode.js";
import { decode, seqOf, terminalRunEventTypes } from "./decode.js";

export type LogPhase = "idle" | "catching-up" | "live" | "gap" | "error";

export interface RunEventsState {
    events: readonly RunEventRecord[];
    contiguousSeq: number;
    maxSeq: number;
    phase: LogPhase;
    error: TransportError | null;
    terminal: boolean;
    version: number;
}

interface RunLog {
    runId: string;
    events: RunEventRecord[];
    seen: Set<number>;
    contiguousSeq: number;
    maxSeq: number;
    phase: LogPhase;
    error: TransportError | null;
    terminal: boolean;
    settled: boolean;
    version: number;
    snapshot: RunEventsState;
    listeners: Set<() => void>;
    pending: Promise<void> | null;
    timer: ReturnType<typeof setTimeout> | null;
    frame: number | null;
    touched: number;
}

export const catchUpLimit = 500;

const catchUpDebounceMs = 150;

const maxLogs = 8;
const maxEvents = 20000;

const logs = new Map<string, RunLog>();

const emptyState: RunEventsState = Object.freeze({
    events: Object.freeze([]) as readonly RunEventRecord[],
    contiguousSeq: 0,
    maxSeq: 0,
    phase: "idle" as LogPhase,
    error: null,
    terminal: false,
    version: 0,
});

let ticket = 0;

function snapshotOf(log: RunLog): RunEventsState {
    return Object.freeze({
        events: log.events.slice(),
        contiguousSeq: log.contiguousSeq,
        maxSeq: log.maxSeq,
        phase: log.phase,
        error: log.error,
        terminal: log.terminal,
        version: log.version,
    });
}

function evict(): void {
    while (logs.size > maxLogs) {
        const idle: RunLog[] = [];
        logs.forEach((held) => {
            if (held.listeners.size === 0) {
                idle.push(held);
            }
        });
        if (idle.length === 0) {
            return;
        }
        let oldest = idle[0];
        for (const held of idle) {
            if (held.touched < oldest.touched) {
                oldest = held;
            }
        }
        if (oldest.timer !== null) {
            clearTimeout(oldest.timer);
        }
        if (oldest.frame !== null) {
            cancelAnimationFrame(oldest.frame);
        }
        logs.delete(oldest.runId);
    }
}

export function ensureLog(runId: string): void {
    log(runId);
}

function log(runId: string): RunLog {
    const held = logs.get(runId);
    if (held !== undefined) {
        ticket += 1;
        held.touched = ticket;
        return held;
    }
    ticket += 1;
    const created: RunLog = {
        runId,
        events: [],
        seen: new Set<number>(),
        contiguousSeq: 0,
        maxSeq: 0,
        phase: "idle",
        error: null,
        terminal: false,
        settled: false,
        version: 0,
        snapshot: emptyState,
        listeners: new Set<() => void>(),
        pending: null,
        timer: null,
        frame: null,
        touched: ticket,
    };
    logs.set(runId, created);
    evict();
    return created;
}

function emit(target: RunLog): void {
    if (target.frame !== null) {
        return;
    }
    target.frame = requestAnimationFrame(() => {
        target.frame = null;
        target.snapshot = snapshotOf(target);
        target.listeners.forEach((listener) => {
            listener();
        });
    });
}

function insert(target: RunLog, record: RunEventRecord): boolean {
    if (target.seen.has(record.seq)) {
        return false;
    }
    target.seen.add(record.seq);
    let low = 0;
    let high = target.events.length;
    while (low < high) {
        const middle = (low + high) >>> 1;
        if (target.events[middle].seq < record.seq) {
            low = middle + 1;
        } else {
            high = middle;
        }
    }
    target.events.splice(low, 0, record);
    if (target.events.length > maxEvents) {
        const dropped = target.events.shift();
        if (dropped !== undefined) {
            target.seen.delete(dropped.seq);
        }
    }
    if (terminalRunEventTypes.has(record.type)) {
        target.terminal = true;
    }
    return true;
}

function advance(target: RunLog): void {
    let next = target.contiguousSeq + 1;
    while (target.seen.has(next)) {
        target.contiguousSeq = next;
        next += 1;
    }
}

function absorb(target: RunLog, rows: readonly RawRunEvent[]): number {
    let changed = 0;
    for (const row of rows) {
        const seq = seqOf(row);
        if (seq === null) {
            continue;
        }
        if (seq > target.maxSeq) {
            target.maxSeq = seq;
        }
        const record = decode(row);
        if (record === null) {
            if (!target.seen.has(seq)) {
                target.seen.add(seq);
                changed += 1;
            }
            continue;
        }
        if (insert(target, record)) {
            changed += 1;
        }
    }
    if (changed > 0) {
        advance(target);
        target.version += 1;
    }
    return changed;
}

export function ingestLive(runId: string, row: RawRunEvent): void {
    const target = log(runId);
    if (absorb(target, [row]) === 0) {
        return;
    }
    if (target.maxSeq > target.contiguousSeq) {
        target.phase = "gap";
        scheduleCatchUp(runId);
    } else if (target.phase === "idle") {
        target.phase = "live";
    }
    emit(target);
}

export function ingestReplay(runId: string, rows: readonly RawRunEvent[]): void {
    const target = log(runId);
    if (absorb(target, rows) === 0) {
        return;
    }
    emit(target);
}

export async function catchUpNow(runId: string): Promise<void> {
    const target = log(runId);
    if (target.settled) {
        return;
    }
    if (target.pending !== null) {
        return target.pending;
    }
    const running = drain(target);
    target.pending = running;
    try {
        await running;
    } finally {
        target.pending = null;
    }
}

async function drain(target: RunLog): Promise<void> {
    target.phase = "catching-up";
    emit(target);
    try {
        for (;;) {
            const since = target.contiguousSeq;
            const answered = await listRunEvents({ runId: target.runId, sinceSeq: since, limit: catchUpLimit });
            const rows: RawRunEvent[] = answered.events ?? [];
            if (rows.length === 0) {
                if (target.terminal && target.contiguousSeq === target.maxSeq) {
                    target.settled = true;
                }
                break;
            }
            ingestReplay(target.runId, rows);
            if (rows.length < catchUpLimit) {
                break;
            }
            if (target.contiguousSeq <= since) {
                break;
            }
        }
        target.error = null;
        target.phase = target.maxSeq > target.contiguousSeq ? "gap" : "live";
    } catch (thrown: unknown) {
        if (isCancellation(thrown)) {
            target.phase = "live";
        } else {
            target.error = failure(thrown);
            target.phase = "error";
        }
    }
    target.version += 1;
    emit(target);
}

export function scheduleCatchUp(runId: string): void {
    const target = log(runId);
    if (target.settled) {
        return;
    }
    if (target.timer !== null) {
        clearTimeout(target.timer);
    }
    target.timer = setTimeout(() => {
        target.timer = null;
        void catchUpNow(runId);
    }, catchUpDebounceMs);
}

export function subscribe(runId: string, listener: () => void): () => void {
    const target = log(runId);
    target.listeners.add(listener);
    return () => {
        target.listeners.delete(listener);
    };
}

export function getSnapshot(runId: string): RunEventsState {
    const held = logs.get(runId);
    return held === undefined ? emptyState : held.snapshot;
}

export function liveRunIds(): string[] {
    const out: string[] = [];
    logs.forEach((held) => {
        if (!held.terminal) {
            out.push(held.runId);
        }
    });
    return out;
}

export function dropAllLogs(): void {
    logs.forEach((held) => {
        if (held.timer !== null) {
            clearTimeout(held.timer);
            held.timer = null;
        }
        if (held.frame !== null) {
            cancelAnimationFrame(held.frame);
            held.frame = null;
        }
        held.events = [];
        held.seen = new Set<number>();
        held.contiguousSeq = 0;
        held.maxSeq = 0;
        held.phase = "idle";
        held.error = null;
        held.terminal = false;
        held.settled = false;
        held.version = 0;
        held.pending = null;
        held.snapshot = emptyState;
        held.listeners.forEach((listener) => {
            listener();
        });
    });
}
