import { describe, expect, it } from "vitest";

import type { RunEventRecord, RunEventType } from "../../data/runs/decode.js";
import { describe as describeEvent, feed, retryNotices, stepTimeline } from "./log-view.js";

let seq = 0;

function event(type: RunEventType, payload: unknown, at = "2026-09-19T10:00:00Z"): RunEventRecord {
    seq += 1;
    return { seq, type, at, payload };
}

function reset(): void {
    seq = 0;
}

describe("retryNotices", () => {
    it("keeps the last retry of each item and its delay", () => {
        reset();
        const events = [
            event("step.retrying", { runId: "r", itemId: "a", step: "judge", attempt: 1, afterMs: 1000 }),
            event("step.retrying", { runId: "r", itemId: "a", step: "judge", attempt: 2, afterMs: 4000 }),
            event("step.retrying", { runId: "r", itemId: "b", step: "publish", attempt: 1, afterMs: 500 }),
        ];
        const notices = retryNotices(events);
        expect(notices.get("a")).toStrictEqual({
            step: "judge",
            attempt: 2,
            afterMs: 4000,
            at: "2026-09-19T10:00:00Z",
        });
        expect(notices.get("b")?.attempt).toBe(1);
    });

    it("clears the notice once the step starts again", () => {
        reset();
        const events = [
            event("step.retrying", { runId: "r", itemId: "a", step: "judge", attempt: 1, afterMs: 1000 }),
            event("step.started", { runId: "r", itemId: "a", step: "judge" }),
        ];
        expect(retryNotices(events).has("a")).toBe(false);
    });
});

describe("stepTimeline", () => {
    it("pairs each start with its finish and keeps the duration the log reports", () => {
        reset();
        const events = [
            event("step.started", { runId: "r", itemId: "a", step: "resolve_context" }, "2026-09-19T10:00:00Z"),
            event("step.done", { runId: "r", itemId: "a", step: "resolve_context", durationMs: 120 }),
            event("step.started", { runId: "r", itemId: "a", step: "generate_body" }, "2026-09-19T10:00:01Z"),
        ];
        const timeline = stepTimeline(events, "a");
        expect(timeline).toHaveLength(2);
        expect(timeline[0].durationMs).toBe(120);
        expect(timeline[0].finishedAt).not.toBeNull();
        expect(timeline[1].finishedAt).toBeNull();
    });

    it("records the failure code of a step that failed", () => {
        reset();
        const events = [
            event("step.started", { runId: "r", itemId: "a", step: "publish" }),
            event("step.failed", { runId: "r", itemId: "a", step: "publish", code: "EXTERNAL", message: "no" }),
        ];
        const timeline = stepTimeline(events, "a");
        expect(timeline[0].code).toBe("EXTERNAL");
        expect(timeline[0].message).toBe("no");
    });

    it("counts a repeated start of the same step as another attempt", () => {
        reset();
        const events = [
            event("step.started", { runId: "r", itemId: "a", step: "judge" }),
            event("step.retrying", { runId: "r", itemId: "a", step: "judge", attempt: 1, afterMs: 100 }),
            event("step.started", { runId: "r", itemId: "a", step: "judge" }),
        ];
        const timeline = stepTimeline(events, "a");
        expect(timeline).toHaveLength(1);
        expect(timeline[0].attempts).toBe(2);
    });

    it("ignores the steps of another item", () => {
        reset();
        const events = [
            event("step.started", { runId: "r", itemId: "b", step: "judge" }),
            event("step.done", { runId: "r", itemId: "b", step: "judge", durationMs: 5 }),
        ];
        expect(stepTimeline(events, "a")).toStrictEqual([]);
    });
});

describe("feed", () => {
    it("answers newest first and stops at the limit", () => {
        reset();
        const events = [
            event("run.queued", { runId: "r", kind: "generate", items: 3 }),
            event("run.started", { runId: "r" }),
            event("item.started", { runId: "r", itemId: "a" }),
        ];
        const entries = feed(events, 2);
        expect(entries.map((entry) => entry.type)).toStrictEqual(["item.started", "run.started"]);
    });

    it("narrows to one item and keeps only entries that name it", () => {
        reset();
        const events = [
            event("run.started", { runId: "r" }),
            event("item.started", { runId: "r", itemId: "a" }),
            event("item.started", { runId: "r", itemId: "b" }),
        ];
        const entries = feed(events, 10, "a");
        expect(entries).toHaveLength(1);
        expect(entries[0].itemId).toBe("a");
    });

    it("reads the retry delay and the attempt off a retry event", () => {
        reset();
        const entry = describeEvent(
            event("step.retrying", { runId: "r", itemId: "a", step: "judge", attempt: 3, afterMs: 8000 }),
        );
        expect(entry.attempt).toBe(3);
        expect(entry.afterMs).toBe(8000);
        expect(entry.step).toBe("judge");
    });

    it("reads why an item paused off its own sentence", () => {
        reset();
        const entry = describeEvent(
            event("item.needs_human", {
                runId: "r",
                itemId: "a",
                reason: "awaiting_parent",
                message: "/guides/x/ waits for its parent /guides/",
            }),
        );
        expect(entry.reason).toBe("awaiting_parent");
        expect(entry.message).toBe("/guides/x/ waits for its parent /guides/");
    });

    it("leaves every field null for a payload it cannot read", () => {
        reset();
        const entry = describeEvent(event("step.done", null));
        expect(entry.step).toBeNull();
        expect(entry.durationMs).toBeNull();
    });
});
