import { describe, expect, it } from "vitest";

import { potteryEdges, potteryEntities } from "./fixture.js";
import { buildGraphIndex } from "./index.js";
import { isolate, lensCounts, lenses, matchedSet } from "./lens.js";

const index = buildGraphIndex(potteryEntities, potteryEdges);

function ids(set: ReadonlySet<string>): string[] {
    return [...set].sort();
}

describe("matchedSet", () => {
    it("matches by lens", () => {
        expect(ids(matchedSet(index, "all", null))).toHaveLength(10);
        expect(ids(matchedSet(index, "noPage", null))).toStrictEqual(["kiln", "travel"]);
        expect(ids(matchedSet(index, "orphan", null))).toStrictEqual(["care", "kiln"]);
        expect(ids(matchedSet(index, "ai", null))).toStrictEqual(["kiln", "travel"]);
        expect(ids(matchedSet(index, "proposed", null))).toStrictEqual(["glazing", "kiln", "mugs", "stone", "teapots"]);
    });

    it("narrows by kind and intersects with the lens", () => {
        expect(ids(matchedSet(index, "all", new Set(["product"])))).toStrictEqual(["m350", "m500", "travel"]);
        expect(ids(matchedSet(index, "noPage", new Set(["product"])))).toStrictEqual(["travel"]);
        expect(ids(matchedSet(index, "all", new Set()))).toHaveLength(10);
    });
});

describe("lensCounts", () => {
    it("counts every lens once", () => {
        expect(lensCounts(index)).toStrictEqual({ all: 10, noPage: 2, proposed: 5, orphan: 2, ai: 2 });
    });

    it("names every lens", () => {
        expect(lenses).toStrictEqual(["all", "noPage", "proposed", "orphan", "ai"]);
    });
});

describe("isolate", () => {
    it("keeps the matches and every placement ancestor", () => {
        expect(ids(isolate(index, new Set(["kiln"])))).toStrictEqual(["glazing", "kiln", "pottery"]);
        expect(ids(isolate(index, new Set(["travel", "care"])))).toStrictEqual(["care", "mugs", "pottery", "travel"]);
        expect(ids(isolate(index, new Set()))).toStrictEqual([]);
    });
});

describe("matchedSet over page states", () => {
    const stated = buildGraphIndex(potteryEntities, potteryEdges, [
        { entityId: "pottery", pageId: "p1", path: "/pottery/", status: "published", work: "", mismatch: false },
        { entityId: "mugs", pageId: "p2", path: "/pottery/mugs/", status: "published", work: "", mismatch: true },
        { entityId: "care", pageId: "p3", path: "/care/", status: "planned", work: "", mismatch: false },
    ]);

    it("keeps only the nodes in the chosen states", () => {
        expect([...matchedSet(stated, "all", null, ["mismatch"])]).toStrictEqual(["mugs"]);
        expect([...matchedSet(stated, "all", null, ["published", "planned"])].sort()).toStrictEqual(["care", "pottery"]);
    });

    it("ignores the state filter when none is chosen", () => {
        expect(matchedSet(stated, "all", null, []).size).toBe(potteryEntities.length);
    });

    it("asks for both the kind and the state", () => {
        expect([...matchedSet(stated, "all", new Set(["hub"]), ["published"])]).toStrictEqual(["pottery"]);
    });
});
