import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { entityTypeLabel, formatScore, scored } from "./labels.js";

describe("formatScore", () => {
    it("shows a computed score to two places", () => {
        expect(formatScore(1)).toBe("1.00");
        expect(formatScore(0.4236)).toBe("0.42");
    });

    it("shows a dash for a score that was never computed", () => {
        expect(formatScore(0)).toBe("—");
    });
});

describe("scored", () => {
    it("answers false while every entity still sits at the stored default", () => {
        expect(scored([{ score: 0 }, { score: 0 }])).toBe(false);
        expect(scored([])).toBe(false);
    });

    it("answers true as soon as one entity carries a rank", () => {
        expect(scored([{ score: 0 }, { score: 0.31 }])).toBe(true);
    });
});

describe("entityTypeLabel", () => {
    it("names the type of the canonical page by the client's own name", () => {
        expect(entityTypeLabel({ id: "t1", name: "Category", preset: "hub" })).toBe("Category");
    });

    it("says so when the canonical page carries no type", () => {
        expect(entityTypeLabel(null)).toBe(copy.pageTypes.untyped);
    });
});
