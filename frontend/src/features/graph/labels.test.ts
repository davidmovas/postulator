import { describe, expect, it } from "vitest";

import { formatScore, scored } from "./labels.js";

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
