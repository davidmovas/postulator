import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { aProgress } from "../../testing/sync.js";
import { pagePrefix, runRowLabel, syncActionLabel } from "./rows.js";

describe("syncActionLabel", () => {
    it("offers to sync the site when no sync is under way", () => {
        expect(syncActionLabel(null)).toBe(copy.palette.syncSite);
    });

    it("says a sync is under way, in which stage, and offers to open it", () => {
        expect(syncActionLabel(aProgress({ stage: "render" }))).toBe("Open the sync under way · visiting the pages");
    });
});

const now = new Date("2026-09-21T10:00:00Z");

describe("runRowLabel", () => {
    const run = {
        id: "b3e009fb-85fa-401c-a349-e0182d01c3fd",
        kind: "generate",
        stats: { items: 5 },
        startedAt: "2026-09-21T09:58:00Z",
        createdAt: "2026-09-21T09:50:00Z",
    };

    it("names the run by what it does, never by its id", () => {
        const label = runRowLabel(run, now);
        expect(label).toContain("5 pages");
        expect(label).not.toContain("b3e009fb");
        expect(label).not.toContain("generate");
    });

    it("dates a run that never started by when it was asked for", () => {
        const queued = { ...run, startedAt: "" };
        expect(runRowLabel(queued, now)).toBe(runRowLabel({ ...run, startedAt: run.createdAt }, now));
    });
});

describe("pagePrefix", () => {
    it("asks for a true path prefix, because the backend has no substring filter", () => {
        expect(pagePrefix("brewing")).toBe("/brewing");
        expect(pagePrefix("/brewing")).toBe("/brewing");
        expect(pagePrefix("  brewing  ")).toBe("/brewing");
        expect(pagePrefix("")).toBe("");
        expect(pagePrefix("   ")).toBe("");
    });
});
