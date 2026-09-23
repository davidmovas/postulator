import { describe, expect, it } from "vitest";

import type { ItemProgress } from "../../data/runs/derive.js";
import type { Run, RunItem } from "../../data/types.js";
import {
    countdown,
    dueMs,
    itemView,
    itemViews,
    remainingMs,
    retryState,
    revertState,
    runView,
    statsView,
    waitingUntil,
} from "./authority.js";
import { recipeSteps, stepPips, stepPosition } from "./recipe.js";

function anItem(overrides: Partial<RunItem> = {}): RunItem {
    return {
        id: "i1",
        runId: "r1",
        siteId: "s1",
        targetId: "p1",
        status: "running",
        currentStep: "generate_body",
        attempts: 1,
        pauseReason: "",
        error: "",
        retryable: true,
        retryBlockedReason: "",
        wakeAt: null,
        createdAt: "2026-09-19T10:00:00Z",
        updatedAt: "2026-09-19T10:00:00Z",
        finishedAt: null,
        ...overrides,
    };
}

function aProgress(overrides: Partial<ItemProgress> = {}): ItemProgress {
    return {
        itemId: "i1",
        step: "judge",
        stepStartedAt: "2026-09-19T10:05:00Z",
        attempt: 2,
        lastError: { code: "EXTERNAL", message: "the judge could not be reached" },
        state: "retrying",
        ...overrides,
    };
}

function aRun(overrides: Partial<Run> = {}): Run {
    return {
        id: "r1",
        siteId: "s1",
        kind: "generate",
        status: "running",
        targets: ["p1"],
        recipe: [
            { name: "resolve_context", enabled: true, params: null },
            { name: "generate_body", enabled: true, params: null },
            { name: "generate_images", enabled: false, params: null },
            { name: "judge", enabled: true, params: null },
        ],
        templateId: "t1",
        templateVersion: 3,
        publishMode: "draft",
        budget: { maxUsd: 5, maxTokens: 0 },
        stats: { items: 4, done: 2, failed: 1, tokens: 1200, usd: 0.4 },
        createdBy: "user",
        parentRunId: null,
        pauseReason: "",
        error: "",
        deadlineAt: "2026-09-19T18:00:00Z",
        createdAt: "2026-09-19T10:00:00Z",
        startedAt: "2026-09-19T10:00:01Z",
        finishedAt: null,
        ...overrides,
    };
}

describe("revertState", () => {
    it.each([
        ["completed", { kind: "ready" }],
        ["failed", { kind: "ready" }],
        ["cancelled", { kind: "ready" }],
        ["running", { kind: "blocked", reason: "in_flight" }],
        ["paused", { kind: "blocked", reason: "in_flight" }],
        ["pending", { kind: "blocked", reason: "in_flight" }],
        ["waiting", { kind: "blocked", reason: "in_flight" }],
    ])("a %s run", (status, want) => {
        expect(revertState(aRun({ status }))).toStrictEqual(want);
    });

    it("refuses a revert of a revert", () => {
        expect(revertState(aRun({ kind: "revert", status: "completed" }))).toStrictEqual({
            kind: "blocked",
            reason: "already_a_revert",
        });
    });
});

describe("itemView", () => {
    it("takes the current step from the log while the run is live", () => {
        const view = itemView(anItem(), aProgress(), false);
        expect(view.step).toBe("judge");
    });

    it("takes the current step from the row once the run is terminal", () => {
        const view = itemView(anItem(), aProgress(), true);
        expect(view.step).toBe("generate_body");
    });

    it("falls back to the row when the log has no step yet", () => {
        const view = itemView(anItem(), aProgress({ step: null }), false);
        expect(view.step).toBe("generate_body");
    });

    it("never takes status, error, attempts or wakeAt from the log", () => {
        const item = anItem({ status: "paused", error: "stopped", attempts: 4, wakeAt: "2026-09-19T11:00:00Z" });
        const view = itemView(item, aProgress({ state: "running", lastError: null }), false);
        expect(view.item.status).toBe("paused");
        expect(view.item.error).toBe("stopped");
        expect(view.item.attempts).toBe(4);
        expect(view.item.wakeAt).toBe("2026-09-19T11:00:00Z");
    });

    it("keeps the per-step failure and retry attempt from the log even when terminal", () => {
        const view = itemView(anItem(), aProgress(), true);
        expect(view.stepAttempt).toBe(2);
        expect(view.stepFailure).toStrictEqual({ code: "EXTERNAL", message: "the judge could not be reached" });
        expect(view.stepStartedAt).toBe("2026-09-19T10:05:00Z");
    });

    it("survives an item the log has never mentioned", () => {
        const view = itemView(anItem(), undefined, false);
        expect(view.step).toBe("generate_body");
        expect(view.stepAttempt).toBe(0);
        expect(view.stepFailure).toBeNull();
    });
});

describe("itemViews", () => {
    it("keeps the order of the rows", () => {
        const rows = [anItem({ id: "a" }), anItem({ id: "b" })];
        const progress = new Map<string, ItemProgress>([["b", aProgress({ itemId: "b", step: "publish" })]]);
        const views = itemViews(rows, progress, false);
        expect(views.map((view) => view.item.id)).toStrictEqual(["a", "b"]);
        expect(views[1].step).toBe("publish");
    });
});

describe("statsView", () => {
    it("reads the row totals, which carry no needsHuman and no calls", () => {
        const view = statsView({ items: 4, done: 2, failed: 1, tokens: 1200, usd: 0.4 });
        expect(view.needsHuman).toBeNull();
        expect(view.calls).toBeNull();
        expect(view.done).toBe(2);
    });

    it("reads the live totals folded from the log", () => {
        const view = statsView({ items: 4, done: 2, failed: 1, needsHuman: 1, tokens: 1200, usd: 0.4, calls: 9 });
        expect(view.needsHuman).toBe(1);
        expect(view.calls).toBe(9);
    });
});

describe("runView", () => {
    it("reports an active run", () => {
        const view = runView(aRun());
        expect(view.active).toBe(true);
        expect(view.terminal).toBe(false);
        expect(view.paused).toBe(false);
    });

    it("names a budget pause separately from any other pause", () => {
        const budget = runView(aRun({ status: "paused", pauseReason: "budget_exceeded" }));
        expect(budget.paused).toBe(true);
        expect(budget.budgetPaused).toBe(true);
        const human = runView(aRun({ status: "paused", pauseReason: "needs_human" }));
        expect(human.paused).toBe(true);
        expect(human.budgetPaused).toBe(false);
    });

    it("treats a zero cap as no cap, which is what the engine does", () => {
        expect(runView(aRun({ budget: { maxUsd: 0, maxTokens: 0 } })).capped).toBe(false);
        expect(runView(aRun()).capped).toBe(true);
    });

    it("reports every terminal status as terminal", () => {
        for (const status of ["completed", "failed", "cancelled"]) {
            expect(runView(aRun({ status })).terminal).toBe(true);
        }
    });
});

describe("retryState", () => {
    it("refuses while the item has not stopped", () => {
        expect(retryState(anItem({ status: "running" }))).toStrictEqual({ kind: "busy" });
        expect(retryState(anItem({ status: "pending" }))).toStrictEqual({ kind: "busy" });
    });

    it("carries the reason code the row declares", () => {
        const blocked = retryState(
            anItem({ status: "completed", retryable: false, retryBlockedReason: "inputs_expired" }),
        );
        expect(blocked).toStrictEqual({ kind: "blocked", reason: "inputs_expired" });
    });

    it("allows a retry on a stopped, retryable item", () => {
        expect(retryState(anItem({ status: "failed" }))).toStrictEqual({ kind: "ready" });
    });
});

describe("waiting and countdowns", () => {
    it("only answers a wake time for a waiting item", () => {
        expect(waitingUntil(anItem({ status: "waiting", wakeAt: "2026-09-19T11:00:00Z" }))).toBe(
            "2026-09-19T11:00:00Z",
        );
        expect(waitingUntil(anItem({ status: "running", wakeAt: "2026-09-19T11:00:00Z" }))).toBeNull();
    });

    it("never counts below zero and refuses unparseable time", () => {
        const now = Date.parse("2026-09-19T11:00:00Z");
        expect(remainingMs("2026-09-19T11:00:30Z", now)).toBe(30_000);
        expect(remainingMs("2026-09-19T10:59:00Z", now)).toBe(0);
        expect(remainingMs("not a time", now)).toBeNull();
        expect(remainingMs(null, now)).toBeNull();
    });

    it("counts a retry from when it was announced", () => {
        const now = Date.parse("2026-09-19T11:00:00Z");
        expect(dueMs("2026-09-19T10:59:50Z", 30_000, now)).toBe(20_000);
        expect(dueMs("not a time", 30_000, now)).toBeNull();
    });

    it("renders minutes and seconds", () => {
        expect(countdown(0)).toBe("0:00");
        expect(countdown(9_000)).toBe("0:09");
        expect(countdown(125_000)).toBe("2:05");
    });
});

describe("recipe", () => {
    it("takes the pipeline order from the run and drops disabled steps", () => {
        expect(recipeSteps(aRun())).toStrictEqual(["resolve_context", "generate_body", "judge"]);
    });

    it("answers nothing for a run with no recipe", () => {
        expect(recipeSteps(aRun({ recipe: null }))).toStrictEqual([]);
        expect(recipeSteps(undefined)).toStrictEqual([]);
    });

    it("numbers a step by its place in the recipe", () => {
        const steps = recipeSteps(aRun());
        expect(stepPosition(steps, "judge")).toBe(3);
        expect(stepPosition(steps, "generate_images")).toBe(0);
        expect(stepPosition(steps, "")).toBe(0);
    });

    it("fills the pips up to and including the current step", () => {
        const steps = recipeSteps(aRun());
        expect(stepPips(steps, "generate_body")).toStrictEqual([true, true, false]);
        expect(stepPips(steps, "")).toStrictEqual([false, false, false]);
    });
});
