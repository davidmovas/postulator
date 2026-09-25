import { describe, expect, it } from "vitest";

import type { RunItem } from "../../data/types.js";
import { holdBody, itemBadge, itemNote, parentRegenerable, primaryAction, queuedAfter, regenerateState } from "./hold.js";

function anItem(overrides: Partial<RunItem> = {}): RunItem {
    return {
        id: "i1",
        runId: "r1",
        siteId: "s1",
        targetId: "p1",
        status: "failed",
        currentStep: "validate",
        seq: 0,
        blockedBy: "",
        attempts: 0,
        pauseReason: "",
        error: "",
        note: "",
        waitingFor: null,
        retryable: true,
        retryBlockedReason: "",
        wakeAt: null,
        createdAt: "2026-09-23T10:00:00Z",
        updatedAt: "2026-09-23T10:00:00Z",
        finishedAt: null,
        ...overrides,
    };
}

const heldChild = anItem({
    status: "paused",
    currentStep: "publish",
    pauseReason: "awaiting_parent",
    note: "/guides/x/ waits for its parent /guides/, which is not on the site yet",
    waitingFor: { pageId: "p0", path: "/guides/", itemId: "i0", itemStatus: "failed", step: "validate" },
});

describe("regenerateState", () => {
    it.each([
        { status: "pending", published: false, want: "busy" },
        { status: "running", published: false, want: "busy" },
        { status: "waiting", published: false, want: "busy" },
        { status: "failed", published: true, want: "published" },
        { status: "failed", published: false, want: "ready" },
        { status: "paused", published: false, want: "ready" },
        { status: "cancelled", published: false, want: "ready" },
    ])("answers $want for a $status item that published: $published", ({ status, published, want }) => {
        expect(regenerateState(anItem({ status }), published).kind).toBe(want);
    });
});

describe("itemBadge", () => {
    it("calls a child held for its parent waiting, not paused", () => {
        const badge = itemBadge(heldChild);
        expect(badge.label).toBe("Waiting");
        expect(badge.tone).toBe("info");
    });

    it("keeps the status of every other item", () => {
        expect(itemBadge(anItem({ status: "paused", pauseReason: "needs_human" })).label).toBe("Paused");
        expect(itemBadge(anItem({ status: "failed" })).tone).toBe("danger");
    });
});

describe("itemNote", () => {
    it("names the parent a held child waits for", () => {
        expect(itemNote(heldChild)).toBe("waits for /guides/");
    });

    it("says what the step said when it paused for anything else", () => {
        expect(itemNote(anItem({ status: "paused", note: "a human edited /x/ on the site" }))).toBe(
            "a human edited /x/ on the site",
        );
    });

    it("says nothing for an item that did not pause", () => {
        expect(itemNote(anItem({ status: "failed", error: "section missing" }))).toBe("");
    });
});

describe("holdBody", () => {
    it.each([
        { itemStatus: "failed", itemId: "i0", want: /failed at Validate in this run.*Regenerate it/ },
        { itemStatus: "paused", itemId: "i0", want: /paused in this run/ },
        { itemStatus: "running", itemId: "i0", want: /being written in this run/ },
        { itemStatus: "", itemId: "", want: /not part of this run/ },
    ])("tells what the parent is doing when it is $itemStatus", ({ itemStatus, itemId, want }) => {
        expect(holdBody({ pageId: "p0", path: "/guides/", itemId, itemStatus, step: "validate" })).toMatch(want);
    });
});

describe("parentRegenerable", () => {
    it("offers to regenerate a parent that stopped in this run", () => {
        expect(parentRegenerable(heldChild.waitingFor)).toBe(true);
    });

    it("does not offer a parent that is still going or not in this run", () => {
        expect(parentRegenerable({ pageId: "p0", path: "/g/", itemId: "i0", itemStatus: "running", step: "" })).toBe(
            false,
        );
        expect(parentRegenerable({ pageId: "p0", path: "/g/", itemId: "", itemStatus: "", step: "" })).toBe(false);
        expect(parentRegenerable(null)).toBe(false);
    });
});

describe("primaryAction", () => {
    it.each([
        { name: "a page held at validate is accepted", item: anItem({ status: "paused", pauseReason: "needs_human", currentStep: "validate" }), want: "accept" },
        { name: "a page held elsewhere for a human is retried", item: anItem({ status: "paused", pauseReason: "needs_human", currentStep: "publish" }), want: "retry" },
        { name: "a page whose writer gave up is regenerated", item: anItem({ status: "failed", currentStep: "generate_body" }), want: "regenerate" },
        { name: "a page that failed elsewhere is retried", item: anItem({ status: "failed", currentStep: "publish" }), want: "retry" },
        { name: "a cancelled page is regenerated", item: anItem({ status: "cancelled" }), want: "regenerate" },
        { name: "a child waiting for its parent points at the parent", item: heldChild, want: "parent" },
        { name: "a running page has nothing to decide", item: anItem({ status: "running" }), want: null },
        { name: "a finished page has nothing to decide", item: anItem({ status: "completed" }), want: null },
    ])("$name", ({ item, want }) => {
        expect(primaryAction(item)).toBe(want);
    });
});

describe("queuedAfter", () => {
    it("names the page a queued item sits behind", () => {
        const queued = anItem({
            status: "pending",
            blockedBy: "i0",
            waitingFor: { pageId: "p0", path: "/guides/", itemId: "i0", itemStatus: "running", step: "publish" },
        });
        expect(queuedAfter(queued)).toBe("/guides/");
    });

    it("says nothing for an item that waits for no one", () => {
        expect(queuedAfter(anItem({ status: "pending" }))).toBe("");
        expect(queuedAfter(anItem({ status: "pending", blockedBy: "i0", waitingFor: null }))).toBe("");
    });
});

describe("a page that finished with something to check", () => {
    const noted = anItem({ status: "completed", currentStep: "report", note: "1 owed link is missing" });

    it("reads as done with a warning and shows what it lacks", () => {
        const badge = itemBadge(noted);
        expect(badge.tone).toBe("warn");
        expect(badge.label).not.toBe(itemBadge(anItem({ status: "completed", currentStep: "report" })).label);
        expect(itemNote(noted)).toBe("1 owed link is missing");
    });

    it("reads a page that finished clean as plainly done", () => {
        const clean = anItem({ status: "completed", currentStep: "report" });
        expect(itemBadge(clean).tone).not.toBe("warn");
        expect(itemNote(clean)).toBe("");
    });
});
