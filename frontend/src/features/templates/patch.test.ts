import { describe, expect, it } from "vitest";

import type { JsonObject } from "../../domain/merge-patch.js";
import { applyPatch, layerAt, layered, patchBetween, patchObject, paths, profilePath, touches } from "./patch.js";
import { draftFromJson } from "./spec.js";

const base: JsonObject = {
    sections: [
        {
            heading: "Overview",
            intent: "Define the topic",
            targetWords: 250,
            required: true,
            keywordRules: { include: [], primaryInHeading: true },
        },
    ],
    tone: "Authoritative and plain",
    length: { min: 1600, max: 2400 },
    keywordRules: { primaryInTitle: true, primaryInH1: true, primaryInFirstParagraph: true, maxDensity: 0.025 },
    linkRules: {
        upDepth: 1,
        downLinks: true,
        siblingMinWeight: 0.5,
        maxLinks: 20,
        maxPerTarget: 1,
        parentLinkWithinParagraphs: 2,
        childrenSection: true,
    },
    metaRules: { titlePattern: "{primaryKeyword}", descriptionMax: 155 },
    images: { featured: true, inline: 0, source: "ai" },
    modelProfiles: {},
    recipe: [
        { name: "resolve_context", enabled: true },
        { name: "generate_body", enabled: true },
    ],
};

describe("applyPatch", () => {
    it("merges an object leaf by leaf", () => {
        expect(applyPatch({ length: { min: 1, max: 2 } }, { length: { min: 5 } })).toStrictEqual({
            length: { min: 5, max: 2 },
        });
    });

    it("replaces an array wholesale", () => {
        expect(applyPatch({ items: [1, 2, 3] }, { items: [9] })).toStrictEqual({ items: [9] });
    });

    it("removes a key given null", () => {
        expect(applyPatch({ a: 1, b: 2 }, { b: null })).toStrictEqual({ a: 1 });
    });

    it("treats a non-object target as empty", () => {
        expect(applyPatch(null, { a: { b: 1 } })).toStrictEqual({ a: { b: 1 } });
    });
});

describe("patchBetween", () => {
    it("answers undefined when nothing was touched", () => {
        const draft = draftFromJson(base);
        expect(patchBetween(draft, draft)).toBeUndefined();
    });

    it("carries only the leaf that changed", () => {
        const draft = draftFromJson(base);
        expect(patchBetween(draft, { ...draft, lengthMin: 600, lengthMax: 900 })).toStrictEqual({
            length: { min: 600, max: 900 },
        });
    });

    it("replaces the recipe wholesale when one step is switched off", () => {
        const draft = draftFromJson(base);
        const recipe = draft.recipe.map((step) =>
            step.name === "generate_body" ? { ...step, enabled: false } : step,
        );
        expect(patchBetween(draft, { ...draft, recipe })).toStrictEqual({
            recipe: [
                { name: "resolve_context", enabled: true },
                { name: "generate_body", enabled: false },
            ],
        });
    });

    it("removes a pinned role with null", () => {
        const pinned = draftFromJson({ ...base, modelProfiles: { writer: { provider: "anthropic", model: "opus" } } });
        expect(patchBetween(pinned, { ...pinned, profiles: [] })).toStrictEqual({ modelProfiles: { writer: null } });
    });

    it("round-trips through apply", () => {
        const draft = draftFromJson(base);
        const edited = { ...draft, lengthMin: 600, lengthMax: 900, downLinks: false };
        const patch = patchObject(patchBetween(draft, edited));
        expect(layered(draft, patch)).toStrictEqual(edited);
    });
});

describe("touches", () => {
    const patch: JsonObject = { length: { min: 600 }, sections: [{ heading: "Only" }] };

    it("finds a leaf the patch sets", () => {
        expect(touches(patch, paths.lengthMin)).toBe(true);
    });

    it("leaves a sibling the patch does not set", () => {
        expect(touches(patch, paths.lengthMax)).toBe(false);
    });

    it("covers everything under a replaced array", () => {
        expect(touches(patch, ["sections", "0", "heading"])).toBe(true);
    });

    it("answers false with no patch at all", () => {
        expect(touches(null, paths.tone)).toBe(false);
    });
});

describe("layerAt", () => {
    const site: JsonObject = { length: { min: 600 }, tone: "Brisk" };
    const page: JsonObject = { tone: "Formal" };

    it("gives the page the last word", () => {
        expect(layerAt(paths.tone, site, page)).toBe("page");
    });

    it("names the site when only the site set it", () => {
        expect(layerAt(paths.lengthMin, site, page)).toBe("site");
    });

    it("falls back to the template", () => {
        expect(layerAt(paths.maxLinks, site, page)).toBe("global");
        expect(layerAt(profilePath("writer"), site, page)).toBe("global");
    });
});
