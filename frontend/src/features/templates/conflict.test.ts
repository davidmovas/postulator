import { describe, expect, it } from "vitest";

import { blankDraft } from "./blank.js";
import type { ConflictKind, Stamp } from "./conflict.js";
import { beneathOf, conflictOf } from "./conflict.js";
import type { Layer } from "./patch.js";

const spec = blankDraft();
const opened: Stamp = { beneath: beneathOf(spec), overrideUpdatedAt: "2026-09-20T10:00:00Z" };
const moved = beneathOf({ ...spec, maxLinks: 3 });

describe("beneathOf", () => {
    it("is the same string for the same spec and a different one when the spec moves", () => {
        expect(beneathOf(spec)).toBe(beneathOf(blankDraft()));
        expect(beneathOf(spec)).not.toBe(moved);
    });

    it("answers for a layer that is not loaded yet", () => {
        expect(beneathOf(null)).toBe("");
    });
});

describe("conflictOf", () => {
    const cases: readonly { name: string; layer: Layer; current: Stamp; dirty: boolean; want: ConflictKind }[] = [
        {
            name: "a clean editor never conflicts",
            layer: "site",
            current: { beneath: moved, overrideUpdatedAt: null },
            dirty: false,
            want: "none",
        },
        { name: "nothing moved, nothing to say", layer: "site", current: opened, dirty: true, want: "none" },
        {
            name: "the spec beneath the layer moved",
            layer: "site",
            current: { ...opened, beneath: moved },
            dirty: true,
            want: "template",
        },
        {
            name: "the template itself moved while it was being edited",
            layer: "global",
            current: { ...opened, beneath: moved },
            dirty: true,
            want: "template",
        },
        {
            name: "a rename does not move the spec, so it is not a conflict",
            layer: "site",
            current: { ...opened },
            dirty: true,
            want: "none",
        },
        {
            name: "the layer's own override was saved elsewhere",
            layer: "site",
            current: { ...opened, overrideUpdatedAt: "2026-09-20T11:00:00Z" },
            dirty: true,
            want: "override",
        },
        {
            name: "the layer's own override was dropped elsewhere",
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
            name: "a moved spec wins over a moved override",
            layer: "site",
            current: { beneath: moved, overrideUpdatedAt: "2026-09-20T11:00:00Z" },
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
        expect(conflictOf("site", null, { beneath: moved, overrideUpdatedAt: null }, true)).toBe("none");
    });
});
