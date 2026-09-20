import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { blankDraft } from "./blank.js";
import { diffRows } from "./diff.js";
import type { SpecDraft } from "./spec.js";

function applied(below: SpecDraft, rows: readonly { revert: Partial<SpecDraft> }[]): SpecDraft {
    return rows.reduce<SpecDraft>((held, row) => ({ ...held, ...row.revert }), below);
}

describe("diffRows", () => {
    it("finds nothing between a draft and itself", () => {
        expect(diffRows(blankDraft(), blankDraft())).toEqual([]);
    });

    it("names a changed rule with both values", () => {
        const below = blankDraft();
        const here: SpecDraft = { ...below, maxLinks: 4 };
        const rows = diffRows(below, here);
        expect(rows).toHaveLength(1);
        expect(rows[0].key).toBe("maxLinks");
        expect(rows[0].label).toBe(copy.templates.links.maxLinks);
        expect(rows[0].below).toBe("8");
        expect(rows[0].here).toBe("4");
    });

    it("shows a density as a percentage, not as a fraction", () => {
        const below = blankDraft();
        const rows = diffRows(below, { ...below, maxDensity: 0.03 });
        expect(rows[0].below).toBe("2%");
        expect(rows[0].here).toBe("3%");
    });

    it("counts the sections rather than listing them", () => {
        const below = blankDraft();
        const here: SpecDraft = { ...below, sections: [...below.sections, ...below.sections] };
        const rows = diffRows(below, here);
        expect(rows.map((row) => row.key)).toEqual(["sections"]);
        expect(rows[0].below).toBe(copy.templates.sectionCount(1));
        expect(rows[0].here).toBe(copy.templates.sectionCount(2));
    });

    it("names a pinned role and says the layer underneath pins nothing", () => {
        const below = blankDraft();
        const here: SpecDraft = {
            ...below,
            profiles: [{ role: "writer", provider: "openai", model: "gpt-5.1" }],
        };
        const rows = diffRows(below, here);
        expect(rows).toHaveLength(1);
        expect(rows[0].key).toBe("role-writer");
        expect(rows[0].below).toBe(copy.templates.layer.none);
        expect(rows[0].here).toBe("openai · gpt-5.1");
    });

    it("names a step that was switched off, with words", () => {
        const below = blankDraft();
        const here: SpecDraft = {
            ...below,
            recipe: below.recipe.map((step) => (step.name === "judge" ? { ...step, enabled: false } : step)),
        };
        const rows = diffRows(below, here);
        expect(rows).toHaveLength(1);
        expect(rows[0].label).toBe(copy.templates.steps.judge);
        expect(rows[0].below).toBe(copy.templates.layer.on);
        expect(rows[0].here).toBe(copy.templates.layer.off);
    });

    it("carries a step setting into the value it shows", () => {
        const below = blankDraft();
        const here: SpecDraft = {
            ...below,
            recipe: below.recipe.map((step) =>
                step.name === "repair_links" ? { ...step, params: { iterations: 4 } } : step,
            ),
        };
        const rows = diffRows(below, here);
        expect(rows).toHaveLength(1);
        expect(rows[0].here).toContain("4");
    });

    it("reverting every row puts the draft back where it started", () => {
        const below = blankDraft();
        const here: SpecDraft = {
            ...below,
            maxLinks: 4,
            tone: "plain",
            featuredImage: true,
            imageSource: "ai",
            profiles: [{ role: "judge", provider: "anthropic", model: "claude-opus-4.1" }],
            recipe: below.recipe.map((step) => (step.name === "publish" ? { ...step, enabled: false } : step)),
            sections: [],
        };
        const rows = diffRows(below, here);
        expect(rows.length).toBeGreaterThan(5);
        expect(diffRows(below, applied(here, rows))).toEqual([]);
    });

    it("reverting one row leaves the others alone", () => {
        const below = blankDraft();
        const here: SpecDraft = { ...below, maxLinks: 4, upDepth: 3 };
        const rows = diffRows(below, here);
        const reverted = { ...here, ...rows.filter((row) => row.key === "maxLinks")[0].revert };
        expect(reverted.maxLinks).toBe(below.maxLinks);
        expect(reverted.upDepth).toBe(3);
    });
});
