import { describe, expect, it } from "vitest";

import { potteryEdges, potteryEntities } from "../model/fixture.js";
import { buildGraphIndex } from "../model/index.js";
import { parentBeingReplaced } from "./move-entity.js";

const index = buildGraphIndex(potteryEntities, potteryEdges);

describe("the parent a move replaces", () => {
    it("is the approved parent the entity sits under today", () => {
        expect(parentBeingReplaced(index, "mugs")).toBe("pottery");
        expect(parentBeingReplaced(index, "m350")).toBe("mugs");
    });

    it("is nothing for a root", () => {
        expect(parentBeingReplaced(index, "pottery")).toBeUndefined();
    });

    it("is nothing when the placement is only proposed", () => {
        expect(index.placementParent.get("kiln")).toBe("glazing");
        expect(parentBeingReplaced(index, "kiln")).toBeUndefined();
    });

    it("is nothing for an entity the graph does not hold", () => {
        expect(parentBeingReplaced(index, "no-such-entity")).toBeUndefined();
    });
});
