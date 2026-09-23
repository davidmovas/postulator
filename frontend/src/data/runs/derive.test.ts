import { describe, expect, it } from "vitest";

import type { RunEventRecord } from "./decode.js";
import { decode } from "./decode.js";
import { itemProgress, liveStats } from "./derive.js";

function event(seq: number, type: string, payload: object): RunEventRecord {
    const decoded = decode({ seq, type, at: "2026-09-23T10:00:00Z", payload });
    if (decoded === null) {
        throw new Error(`${type} did not decode`);
    }
    return decoded;
}

describe("itemProgress", () => {
    it("starts a regenerated item over and forgets the failure it had", () => {
        const events = [
            event(1, "item.started", { runId: "r", itemId: "i" }),
            event(2, "step.started", { runId: "r", itemId: "i", step: "validate" }),
            event(3, "item.failed", { runId: "r", itemId: "i", code: "INVALID", message: "section missing" }),
            event(4, "item.restarted", { runId: "r", itemId: "i" }),
        ];

        const held = itemProgress(events).get("i");

        expect(held?.state).toBe("pending");
        expect(held?.lastError).toBeNull();
        expect(held?.step).toBeNull();
    });

    it("holds a child that waits for its parent apart from one that needs a human", () => {
        const events = [
            event(1, "item.needs_human", { runId: "r", itemId: "child", reason: "awaiting_parent", message: "waits" }),
            event(2, "item.needs_human", { runId: "r", itemId: "drift", reason: "needs_human", message: "edited" }),
        ];

        const progress = itemProgress(events);

        expect(progress.get("child")?.state).toBe("held");
        expect(progress.get("drift")?.state).toBe("needs_human");
    });
});

describe("liveStats", () => {
    it("counts every item by where it stands now, so a regenerated failure is not counted twice", () => {
        const events = [
            event(1, "run.queued", { runId: "r", kind: "generate", items: 3 }),
            event(2, "item.failed", { runId: "r", itemId: "parent", code: "INVALID", message: "section missing" }),
            event(3, "item.needs_human", { runId: "r", itemId: "child", reason: "awaiting_parent", message: "waits" }),
            event(4, "item.restarted", { runId: "r", itemId: "parent" }),
            event(5, "item.done", { runId: "r", itemId: "parent" }),
            event(6, "item.done", { runId: "r", itemId: "child" }),
            event(7, "item.needs_human", { runId: "r", itemId: "other", reason: "needs_human", message: "edited" }),
        ];

        const stats = liveStats(events);

        expect(stats.items).toBe(3);
        expect(stats.done).toBe(2);
        expect(stats.failed).toBe(0);
        expect(stats.needsHuman).toBe(1);
        expect(stats.waitingParent).toBe(0);
    });

    it("counts the children still waiting for a parent", () => {
        const stats = liveStats([
            event(1, "run.queued", { runId: "r", kind: "generate", items: 2 }),
            event(2, "item.failed", { runId: "r", itemId: "parent", code: "INVALID", message: "section missing" }),
            event(3, "item.needs_human", { runId: "r", itemId: "child", reason: "awaiting_parent", message: "waits" }),
        ]);

        expect(stats.failed).toBe(1);
        expect(stats.waitingParent).toBe(1);
        expect(stats.needsHuman).toBe(0);
    });
});
