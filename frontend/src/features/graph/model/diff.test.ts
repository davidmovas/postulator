import { describe, expect, it } from "vitest";

import { diffIndex } from "./diff.js";
import { edge, entity, potteryEdges, potteryEntities } from "./fixture.js";
import { buildGraphIndex } from "./index.js";

const before = buildGraphIndex(potteryEntities, potteryEdges);

describe("diffIndex", () => {
    it("reports nothing when nothing changed", () => {
        const again = buildGraphIndex(potteryEntities, potteryEdges);
        expect(diffIndex(before, again)).toStrictEqual({ added: [], removed: [], changedEdges: [], touched: [] });
    });

    it("names the entities that appeared and the ends of the edges that changed", () => {
        const after = buildGraphIndex(
            [...potteryEntities, entity({ id: "saucers", name: "Saucers" })],
            [
                ...potteryEdges.filter((held) => held.id !== "e10"),
                edge("e10", "mugs", "teapots", "related", "approved", 0.86),
                edge("e13", "saucers", "teapots", "parent", "proposed"),
            ],
        );
        const diff = diffIndex(before, after);
        expect(diff.added).toStrictEqual(["saucers"]);
        expect(diff.removed).toStrictEqual([]);
        expect(diff.changedEdges).toStrictEqual(["e10", "e13"]);
        expect([...diff.touched].sort()).toStrictEqual(["mugs", "saucers", "teapots"]);
    });

    it("names the entities that went away", () => {
        const after = buildGraphIndex(
            potteryEntities.filter((held) => held.id !== "care"),
            potteryEdges.filter((held) => held.id !== "e12"),
        );
        const diff = diffIndex(before, after);
        expect(diff.removed).toStrictEqual(["care"]);
        expect(diff.touched).toStrictEqual([]);
    });

    it("treats a changed name or score as a touch", () => {
        const after = buildGraphIndex(
            potteryEntities.map((held) => (held.id === "glazing" ? { ...held, score: 0.75 } : held)),
            potteryEdges,
        );
        expect(diffIndex(before, after).touched).toStrictEqual(["glazing"]);
    });
});
