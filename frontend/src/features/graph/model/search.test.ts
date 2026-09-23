import { describe, expect, it } from "vitest";

import { potteryEntities } from "./fixture.js";
import { rank } from "./search.js";

describe("rank", () => {
    it("puts name prefixes before word starts before keywords", () => {
        const hits = rank(potteryEntities, "mug", 10);
        expect(hits.map((hit) => hit.id)).toStrictEqual(["m350", "m500", "mugs", "travel", "care", "stone"]);
        expect(hits[0].field).toBe("name");
        expect(hits[4].field).toBe("primaryKeyword");
        expect(hits[5].field).toBe("secondaryKeyword");
    });

    it("ignores case and surrounding space", () => {
        expect(rank(potteryEntities, "  GLAZ ", 10).map((hit) => hit.id)).toStrictEqual(["glazing"]);
    });

    it("answers nothing for an empty query", () => {
        expect(rank(potteryEntities, "   ", 10)).toStrictEqual([]);
    });

    it("respects the limit", () => {
        expect(rank(potteryEntities, "mug", 2).map((hit) => hit.id)).toStrictEqual(["m350", "m500"]);
    });

    it("matches inside a name below a word start", () => {
        expect(rank(potteryEntities, "ware", 10).map((hit) => hit.id)).toStrictEqual(["stone"]);
        expect(rank(potteryEntities, "ware", 10)[0].field).toBe("name");
    });
});
