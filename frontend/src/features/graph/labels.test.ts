import { describe, expect, it } from "vitest";

import { entityKinds } from "../../generated/vocab.js";
import { StarShineIcon } from "../../ui/index.js";
import { entityIcon, formatScore, kindLabel, scored } from "./labels.js";

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

describe("the entity kinds in words and icons", () => {
    it("names every entity kind from one source", () => {
        for (const kind of entityKinds) {
            expect(kindLabel(kind)).not.toBe("");
            expect(kindLabel(kind)).not.toBe(kind);
        }
    });

    it("shows a kind it has never heard of as it arrived", () => {
        expect(kindLabel("galaxy")).toBe("galaxy");
        expect(kindLabel("")).toBe("");
    });

    it("draws every entity kind with its own icon", () => {
        const icons = new Set(entityKinds.map((kind) => entityIcon(kind)));
        expect(icons.size).toBe(entityKinds.length);
    });

    it("draws a kind it has never heard of as a custom one", () => {
        expect(entityIcon("galaxy")).toBe(StarShineIcon);
        expect(entityIcon("custom")).toBe(StarShineIcon);
    });
});
