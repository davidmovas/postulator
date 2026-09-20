import { describe, expect, it } from "vitest";

import type { ConflictKind, Stamp } from "./conflict.js";
import { conflictOf, stampsAgree } from "./conflict.js";
import type { Layer } from "./patch.js";

const opened: Stamp = { version: 3, updatedAt: "2026-09-20T10:00:00Z", overrideUpdatedAt: "2026-09-20T10:00:00Z" };

describe("conflictOf", () => {
    const cases: readonly { name: string; layer: Layer; current: Stamp; dirty: boolean; want: ConflictKind }[] = [
        { name: "a clean editor never conflicts", layer: "site", current: { ...opened, version: 9 }, dirty: false, want: "none" },
        { name: "nothing moved, nothing to say", layer: "site", current: opened, dirty: true, want: "none" },
        {
            name: "a bumped version is the template moving underneath",
            layer: "site",
            current: { ...opened, version: 4 },
            dirty: true,
            want: "template",
        },
        {
            name: "a new updatedAt with the same version still counts",
            layer: "global",
            current: { ...opened, updatedAt: "2026-09-20T11:00:00Z" },
            dirty: true,
            want: "template",
        },
        {
            name: "the layer's own override moving is its own kind",
            layer: "site",
            current: { ...opened, overrideUpdatedAt: "2026-09-20T11:00:00Z" },
            dirty: true,
            want: "override",
        },
        {
            name: "the page layer watches its own override too",
            layer: "page",
            current: { ...opened, overrideUpdatedAt: null },
            dirty: true,
            want: "override",
        },
        {
            name: "the global layer has no override to conflict with",
            layer: "global",
            current: { ...opened, overrideUpdatedAt: "2026-09-20T11:00:00Z" },
            dirty: true,
            want: "none",
        },
        {
            name: "the template moving wins over the override moving",
            layer: "site",
            current: { version: 4, updatedAt: "2026-09-20T11:00:00Z", overrideUpdatedAt: "2026-09-20T11:00:00Z" },
            dirty: true,
            want: "template",
        },
    ];

    for (const held of cases) {
        it(held.name, () => {
            expect(conflictOf(held.layer, opened, held.current, held.dirty)).toBe(held.want);
        });
    }

    it("says nothing before the editor has taken a stamp", () => {
        expect(conflictOf("site", null, opened, true)).toBe("none");
    });
});

describe("stampsAgree", () => {
    it("compares all three fields", () => {
        expect(stampsAgree(opened, { ...opened })).toBe(true);
        expect(stampsAgree(opened, { ...opened, version: 4 })).toBe(false);
        expect(stampsAgree(opened, { ...opened, updatedAt: "2026-09-20T11:00:00Z" })).toBe(false);
        expect(stampsAgree(opened, { ...opened, overrideUpdatedAt: null })).toBe(false);
    });
});
