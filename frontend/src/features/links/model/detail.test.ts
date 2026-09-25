import { describe, expect, it } from "vitest";

import { groups } from "./detail.js";
import { mugsDetail, plannedDetail, required } from "./fixture.js";

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
        expect(groups(mugsDetail).counts).toStrictEqual({
            satisfied: 2,
            missing: 2,
            missingRequired: 1,
            blocked: 1,
            offGraph: 3,
            anchorNotAllowed: 1,
            pending: 0,
            unpublished: 0,
        });
    });

    it("counts what waits apart from what is missing", () => {
        expect(groups(plannedDetail).counts).toMatchObject({ satisfied: 0, missing: 0, missingRequired: 0, pending: 3 });

        const waiting = groups({
            ...mugsDetail,
            required: [
                required({ relation: "down", targetEntityId: "cups", state: "awaiting_target", targetOnSite: false }),
                required({ relation: "down", targetEntityId: "bowls", satisfied: true, anchorAllowed: true, state: "target_unpublished", targetOnSite: false }),
            ],
        });
        expect(waiting.counts).toMatchObject({ satisfied: 1, missing: 0, pending: 1, unpublished: 1 });
    });

    it("handles an empty detail", () => {
        const grouped = groups({ ...mugsDetail, required: null, extra: null });
        expect(grouped.up).toStrictEqual([]);
        expect(grouped.extra).toStrictEqual([]);
        expect(grouped.counts.offGraph).toBe(0);
    });
});
