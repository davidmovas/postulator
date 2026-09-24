import { beforeEach, describe, expect, test, vi } from "vitest";

const { fetchEvents } = vi.hoisted(() => ({ fetchEvents: vi.fn() }));

vi.mock("../endpoints/runs.js", () => ({ listRunEvents: fetchEvents }));

import type { RawRunEvent } from "./decode.js";
import { runEventTypes } from "./decode.js";
import {
    catchUpLimit,
    catchUpNow,
    dropAllLogs,
    getSnapshot,
    ingestLive,
    ingestReplay,
    scheduleCatchUp,
} from "./log.js";

function row(seq: number, type = "step.done"): RawRunEvent {
    return { seq, type, at: "2026-09-19T10:00:00Z", payload: { runId: "r", itemId: "i", step: "publish" } };
}

function tick(): Promise<void> {
    return new Promise((resolve) => {
        setTimeout(resolve, 0);
    });
}

beforeEach(() => {
    dropAllLogs();
    fetchEvents.mockReset();
    fetchEvents.mockResolvedValue({ events: [] });
});

describe("the run event log", () => {
    test("sorts out-of-order arrivals and ignores duplicates", async () => {
        ingestReplay("out-of-order", [row(3), row(1), row(2), row(1), row(3)]);
        await tick();

        const state = getSnapshot("out-of-order");
        expect(state.events.map((held) => held.seq)).toEqual([1, 2, 3]);
        expect(state.contiguousSeq).toBe(3);
        expect(state.maxSeq).toBe(3);
    });

    test("drops an unknown event type while keeping seq contiguity", async () => {
        ingestReplay("unknown", [row(1), { seq: 2, type: "run.telepathy", at: "", payload: {} }, row(3)]);
        await tick();

        const state = getSnapshot("unknown");
        expect(state.events.map((held) => held.seq)).toEqual([1, 3]);
        expect(state.contiguousSeq).toBe(3);
    });

    test("keeps a model call out of the log, because ListEvents never replays one", async () => {
        ingestReplay("usage", [
            row(1),
            {
                seq: 2,
                type: "llm.usage",
                at: "2026-09-19T10:00:01Z",
                payload: { runId: "r", itemId: "i", provider: "openai", model: "m", promptTokens: 7, completionTokens: 9, usd: 0.01 },
            },
        ]);
        await tick();

        const state = getSnapshot("usage");
        expect(state.events.map((held) => held.type)).toEqual(["step.done"]);
        expect(runEventTypes).not.toContain("llm.usage");
    });

    test("heals a real gap by asking from contiguousSeq, never maxSeq", async () => {
        ingestLive("gap", row(1));
        ingestLive("gap", row(3));
        await tick();

        const gapped = getSnapshot("gap");
        expect(gapped.contiguousSeq).toBe(1);
        expect(gapped.maxSeq).toBe(3);
        expect(gapped.phase).toBe("gap");

        fetchEvents.mockReset();
        fetchEvents.mockResolvedValueOnce({ events: [row(2), row(3)] });
        fetchEvents.mockResolvedValue({ events: [] });

        await catchUpNow("gap");
        await tick();

        expect(fetchEvents.mock.calls[0][0]).toEqual({ runId: "gap", sinceSeq: 1, limit: catchUpLimit });
        const healed = getSnapshot("gap");
        expect(healed.contiguousSeq).toBe(3);
        expect(healed.maxSeq).toBe(3);
        expect(healed.phase).toBe("live");
    });

    test("runs one catch-up at a time per run", async () => {
        let release: ((value: { events: RawRunEvent[] }) => void) | null = null;
        fetchEvents.mockImplementationOnce(
            () =>
                new Promise<{ events: RawRunEvent[] }>((resolve) => {
                    release = resolve;
                }),
        );

        const first = catchUpNow("single-flight");
        const second = catchUpNow("single-flight");
        expect(release).not.toBeNull();
        (release as unknown as (value: { events: RawRunEvent[] }) => void)({ events: [] });
        await Promise.all([first, second]);

        expect(fetchEvents).toHaveBeenCalledTimes(1);
    });

    test("stops when a full page makes no forward progress", async () => {
        const stuck = Array.from({ length: catchUpLimit }, (_held, index) => row(index + 2));
        fetchEvents.mockResolvedValue({ events: stuck });

        await catchUpNow("stuck");
        await tick();

        expect(fetchEvents).toHaveBeenCalledTimes(1);
        expect(getSnapshot("stuck").contiguousSeq).toBe(0);
        expect(getSnapshot("stuck").maxSeq).toBe(catchUpLimit + 1);
    });

    test("freezes a terminal run once the last catch-up comes back empty", async () => {
        fetchEvents.mockResolvedValueOnce({ events: [row(1, "run.queued"), row(2, "run.completed")] });
        fetchEvents.mockResolvedValue({ events: [] });

        await catchUpNow("terminal");
        await catchUpNow("terminal");
        await tick();

        expect(fetchEvents).toHaveBeenCalledTimes(2);
        expect(getSnapshot("terminal").terminal).toBe(true);

        await catchUpNow("terminal");
        scheduleCatchUp("terminal");
        await tick();

        expect(fetchEvents).toHaveBeenCalledTimes(2);
    });

    test("goes live again when a finished run is resumed or regenerated", async () => {
        fetchEvents.mockResolvedValueOnce({ events: [row(1, "run.queued"), row(2, "run.completed")] });
        fetchEvents.mockResolvedValue({ events: [] });

        await catchUpNow("revived");
        await catchUpNow("revived");
        await tick();
        expect(getSnapshot("revived").terminal).toBe(true);

        ingestLive("revived", row(3, "run.resumed"));
        await tick();
        expect(getSnapshot("revived").terminal).toBe(false);

        fetchEvents.mockReset();
        fetchEvents.mockResolvedValueOnce({ events: [row(4, "step.started"), row(5, "run.completed")] });
        fetchEvents.mockResolvedValue({ events: [] });
        await catchUpNow("revived");
        await tick();

        expect(fetchEvents).toHaveBeenCalled();
        expect(getSnapshot("revived").terminal).toBe(true);
        expect(getSnapshot("revived").contiguousSeq).toBe(5);
    });

    test("does not call a run finished because a late terminal event arrived before its restart", async () => {
        ingestReplay("late", [row(3, "run.resumed"), row(2, "run.completed"), row(1, "run.queued")]);
        await tick();

        expect(getSnapshot("late").terminal).toBe(false);
    });

    test("keeps one frozen snapshot identity between emissions", async () => {
        ingestReplay("identity", [row(1)]);
        await tick();

        const first = getSnapshot("identity");
        expect(getSnapshot("identity")).toBe(first);
        expect(Object.isFrozen(first)).toBe(true);

        ingestReplay("identity", [row(2)]);
        await tick();

        expect(getSnapshot("identity")).not.toBe(first);
    });
});
