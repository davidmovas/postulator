import type { ItemProgress, LiveStats, StepFailure } from "../../data/runs/derive.js";
import type { Run, RunItem, RunTotals } from "../../data/types.js";
import { activeRunStatuses, terminalRunStatuses } from "../../generated/vocab.js";
import {
    itemPending,
    itemRunning,
    kindRevert,
    pauseBudgetExceeded,
    statusPaused,
    statusWaiting,
} from "./statuses.js";

export interface ItemView {
    item: RunItem;
    step: string;
    stepStartedAt: string | null;
    stepAttempt: number;
    stepFailure: StepFailure | null;
}

export function itemView(
    item: RunItem,
    progress: ItemProgress | undefined,
    terminal: boolean,
): ItemView {
    const logged = progress ?? null;
    const step = terminal || logged === null || logged.step === null ? item.currentStep : logged.step;
    return {
        item,
        step,
        stepStartedAt: logged?.stepStartedAt ?? null,
        stepAttempt: logged?.attempt ?? 0,
        stepFailure: logged?.lastError ?? null,
    };
}

export function itemViews(
    items: readonly RunItem[],
    progress: ReadonlyMap<string, ItemProgress>,
    terminal: boolean,
): readonly ItemView[] {
    return items.map((item) => itemView(item, progress.get(item.id), terminal));
}

export interface StatsView {
    items: number;
    done: number;
    failed: number;
    tokens: number;
    usd: number;
    needsHuman: number | null;
    calls: number | null;
}

function isLive(stats: RunTotals | LiveStats): stats is LiveStats {
    return "needsHuman" in stats && "calls" in stats;
}

export function statsView(stats: RunTotals | LiveStats): StatsView {
    return {
        items: stats.items,
        done: stats.done,
        failed: stats.failed,
        tokens: stats.tokens,
        usd: stats.usd,
        needsHuman: isLive(stats) ? stats.needsHuman : null,
        calls: isLive(stats) ? stats.calls : null,
    };
}

export interface RunView {
    status: string;
    pauseReason: string;
    error: string;
    active: boolean;
    terminal: boolean;
    paused: boolean;
    budgetPaused: boolean;
    cap: number;
    capped: boolean;
}

export function runView(run: Run): RunView {
    const paused = run.status === statusPaused;
    const cap = run.budget.maxUsd;
    return {
        status: run.status,
        pauseReason: run.pauseReason,
        error: run.error,
        active: (activeRunStatuses as readonly string[]).includes(run.status),
        terminal: (terminalRunStatuses as readonly string[]).includes(run.status),
        paused,
        budgetPaused: paused && run.pauseReason === pauseBudgetExceeded,
        cap,
        capped: Number.isFinite(cap) && cap > 0,
    };
}

export type RevertBlock = "in_flight" | "already_a_revert";

export type RevertState = { kind: "ready" } | { kind: "blocked"; reason: RevertBlock };

export function revertState(run: Run): RevertState {
    if (run.kind === kindRevert) {
        return { kind: "blocked", reason: "already_a_revert" };
    }
    if (!(terminalRunStatuses as readonly string[]).includes(run.status)) {
        return { kind: "blocked", reason: "in_flight" };
    }
    return { kind: "ready" };
}

export type RetryState =
    | { kind: "ready" }
    | { kind: "busy" }
    | { kind: "blocked"; reason: string };

export function retryState(item: RunItem): RetryState {
    if (item.status === itemRunning || item.status === itemPending) {
        return { kind: "busy" };
    }
    if (!item.retryable) {
        return { kind: "blocked", reason: item.retryBlockedReason };
    }
    return { kind: "ready" };
}

export function waitingUntil(item: RunItem): string | null {
    return item.status === statusWaiting ? item.wakeAt : null;
}

export function remainingMs(at: string | null, now: number): number | null {
    if (at === null || at === "") {
        return null;
    }
    const parsed = Date.parse(at);
    if (Number.isNaN(parsed)) {
        return null;
    }
    return Math.max(parsed - now, 0);
}

export function dueMs(at: string, afterMs: number, now: number): number | null {
    const started = Date.parse(at);
    if (Number.isNaN(started)) {
        return null;
    }
    return Math.max(started + afterMs - now, 0);
}

export function countdown(milliseconds: number): string {
    const total = Math.round(milliseconds / 1000);
    const minutes = Math.floor(total / 60);
    const seconds = total - minutes * 60;
    return `${minutes}:${seconds < 10 ? "0" : ""}${seconds}`;
}
