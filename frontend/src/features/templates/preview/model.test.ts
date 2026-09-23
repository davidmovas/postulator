import { describe, expect, it } from "vitest";

import { blankDraft } from "../blank.js";
import type { SpecDraft } from "../spec.js";
import { imageSlots, skeletonOf } from "./model.js";

function draft(patch: Partial<SpecDraft>): SpecDraft {
    return { ...blankDraft(), ...patch };
}

const sections = [
    { heading: "Overview", intent: "", targetWords: 200, required: true, include: [], primaryInHeading: true },
    { heading: "Steps", intent: "", targetWords: 600, required: true, include: [], primaryInHeading: false },
    { heading: "", intent: "", targetWords: 200, required: false, include: [], primaryInHeading: false },
];

describe("skeletonOf", () => {
    it("sizes every section by its share of the words", () => {
        const skeleton = skeletonOf(draft({ sections, lengthMin: 800, lengthMax: 1200 }));
        expect(skeleton.totalWords).toBe(1000);
        expect(skeleton.sections.map((section) => section.share)).toStrictEqual([0.2, 0.6, 0.2]);
        expect(skeleton.sections[2]?.heading).toBe("");
        expect(skeleton.lengthState).toBe("within");
    });

    it("says when the sections add up to less or more than the page length allows", () => {
        expect(skeletonOf(draft({ sections, lengthMin: 1200, lengthMax: 1500 })).lengthState).toBe("under");
        expect(skeletonOf(draft({ sections, lengthMin: 300, lengthMax: 600 })).lengthState).toBe("over");
        expect(skeletonOf(draft({ sections, lengthMin: 0, lengthMax: 0 })).lengthState).toBe("unbounded");
    });

    it("carries the link, keyword, image and meta rules the page will follow", () => {
        const skeleton = skeletonOf(
            draft({
                sections,
                featuredImage: true,
                inlineImages: 2,
                imageSource: "ai",
                parentLinkWithinParagraphs: 2,
                childrenSection: true,
                maxLinks: 8,
                upDepth: 1,
                titlePattern: "{primaryKeyword} | {siteName}",
                descriptionMax: 155,
                primaryInH1: true,
            }),
        );
        expect(skeleton.featured).toBe(true);
        expect(skeleton.inlineImages).toBe(2);
        expect(skeleton.parentLinkWithin).toBe(2);
        expect(skeleton.childrenSection).toBe(true);
        expect(skeleton.maxLinks).toBe(8);
        expect(skeleton.titleParts).toStrictEqual([
            { kind: "placeholder", text: "primaryKeyword" },
            { kind: "text", text: " | " },
            { kind: "placeholder", text: "siteName" },
        ]);
        expect(skeleton.descriptionMax).toBe(155);
        expect(skeleton.h1Keyword).toBe(true);
    });

    it("handles an empty template without dividing by zero", () => {
        const skeleton = skeletonOf(draft({ sections: [] }));
        expect(skeleton.sections).toStrictEqual([]);
        expect(skeleton.totalWords).toBe(0);
    });
});

describe("imageSlots", () => {
    it("spreads the inline images across the sections and never past the last one", () => {
        expect(imageSlots(3, 0)).toStrictEqual([]);
        expect(imageSlots(3, 1)).toStrictEqual([1]);
        expect(imageSlots(3, 2)).toStrictEqual([0, 1]);
        expect(imageSlots(4, 2)).toStrictEqual([1, 2]);
        expect(imageSlots(2, 5)).toStrictEqual([0, 1]);
        expect(imageSlots(0, 2)).toStrictEqual([]);
    });
});
