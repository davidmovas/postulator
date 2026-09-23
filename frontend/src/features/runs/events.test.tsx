import { describe, expect, it } from "vitest";

import { detailOf } from "./events.js";
import type { FeedEntry } from "./log-view.js";

function anEntry(overrides: Partial<FeedEntry> = {}): FeedEntry {
    return {
        seq: 1,
        type: "item.needs_human",
        at: "2026-09-23T10:00:00Z",
        itemId: "i1",
        step: null,
        code: null,
        message: null,
        reason: null,
        durationMs: null,
        attempt: null,
        afterMs: null,
        usd: null,
        items: null,
        ...overrides,
    };
}

describe("detailOf", () => {
    it("says why an item paused in words, never the stored reason", () => {
        const detail = detailOf(anEntry({ reason: "awaiting_parent", message: "/a/b/ waits for its parent /a/" }));
        expect(detail).toBe("waiting for parent · /a/b/ waits for its parent /a/");
        expect(detail).not.toMatch(/_/);
    });
});
