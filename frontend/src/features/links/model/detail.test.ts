import { describe, expect, it } from "vitest";

import { groups } from "./detail.js";
import { mugsDetail } from "./fixture.js";

describe("groups", () => {
    it("splits the required links by relation and keeps the blocked ones apart", () => {
        const grouped = groups(mugsDetail);
        expect(grouped.up.map((link) => link.targetEntityId)).toStrictEqual(["pottery"]);
        expect(grouped.down.map((link) => link.targetEntityId)).toStrictEqual(["m350", "m500"]);
        expect(grouped.sibling.map((link) => link.targetEntityId)).toStrictEqual(["glazing"]);
        expect(grouped.blocked.map((link) => link.targetEntityId)).toStrictEqual(["travel"]);
    });

    it("splits the extra links by class in a fixed order", () => {
        const grouped = groups(mugsDetail);
        expect(grouped.extra.map((entry) => entry.kind)).toStrictEqual(["unknown_internal", "self", "external"]);
        expect(grouped.extra[0].links.map((link) => link.anchor)).toStrictEqual(["diary"]);
    });

    it("counts what the page owes and carries", () => {
        expect(groups(mugsDetail).counts).toStrictEqual({ satisfied: 2, missing: 2, missingRequired: 1, blocked: 1, offGraph: 3, anchorNotAllowed: 1 });
    });

    it("handles an empty detail", () => {
        const grouped = groups({ ...mugsDetail, required: null, extra: null });
        expect(grouped.up).toStrictEqual([]);
        expect(grouped.extra).toStrictEqual([]);
        expect(grouped.counts.offGraph).toBe(0);
    });
});
