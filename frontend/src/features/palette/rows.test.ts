import { describe, expect, it } from "vitest";

import { pagePrefix, runRowLabel } from "./rows.js";

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
