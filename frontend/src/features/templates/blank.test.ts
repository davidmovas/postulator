import { describe, expect, it } from "vitest";

import { perKindStepNames, stepNames } from "../../generated/vocab.js";
import { blankDraft } from "./blank.js";
import { draftFromJson, specJsonOf } from "./spec.js";

describe("blankDraft", () => {
    const draft = blankDraft();

    it("starts with one required section that has a heading", () => {
        expect(draft.sections).toHaveLength(1);
        expect(draft.sections[0]?.heading).not.toBe("");
        expect(draft.sections[0]?.required).toBe(true);
    });

    it("satisfies every range the Go validator checks", () => {
        expect(draft.lengthMin).toBeGreaterThanOrEqual(0);
        expect(draft.lengthMax).toBeGreaterThanOrEqual(draft.lengthMin);
        expect(draft.maxDensity).toBeGreaterThanOrEqual(0);
        expect(draft.maxDensity).toBeLessThanOrEqual(1);
        expect(draft.upDepth).toBeGreaterThanOrEqual(0);
        expect(draft.siblingMinWeight).toBeGreaterThanOrEqual(0);
        expect(draft.siblingMinWeight).toBeLessThanOrEqual(1);
        expect(draft.maxLinks).toBeGreaterThanOrEqual(0);
        expect(draft.maxPerTarget).toBeGreaterThanOrEqual(0);
        expect(draft.parentLinkWithinParagraphs).toBeGreaterThanOrEqual(0);
        expect(draft.descriptionMax).toBeGreaterThanOrEqual(0);
        expect(draft.inlineImages).toBe(0);
        expect(draft.featuredImage).toBe(false);
        expect(draft.imageSource).toBe("wpmedia");
        expect(draft.profiles).toStrictEqual([]);
    });

    it("declares every shipped step, all on but the image step", () => {
        expect(draft.recipe.map((step) => step.name)).toStrictEqual([...stepNames]);
        expect(draft.recipe.every((step) => step.declared)).toBe(true);
        for (const step of draft.recipe) {
            expect(step.enabled).toBe(step.name !== "generate_images");
        }
    });

    it("names no step a run kind owns, which runs.Start would refuse", () => {
        expect(perKindStepNames.length).toBeGreaterThan(0);
        for (const owned of perKindStepNames) {
            expect(draft.recipe.map((step) => step.name)).not.toContain(owned);
        }
    });

    it("round trips through the JSON form", () => {
        expect(draftFromJson(specJsonOf(draft))).toStrictEqual(draft);
    });
});
