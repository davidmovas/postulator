import { describe, expect, it } from "vitest";

import type { RunEventRecord } from "./decode.js";
import { decode } from "./decode.js";
import { itemProgress } from "./derive.js";

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
});
