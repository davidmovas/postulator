import { afterEach, describe, expect, it } from "vitest";

import type { ProposalOpener } from "./proposal-memory.js";
import { blankMemory, forgetProposal, memoryOf, remember, seedKey } from "./proposal-memory.js";

const openers: readonly ProposalOpener[] = ["pages", "keywords", "seed"];

afterEach(() => {
    for (const siteId of ["s1", "s2"]) {
        for (const opener of openers) {
            forgetProposal(siteId, opener);
        }
    }
});

describe("the proposal memory", () => {
    it.each([
        { opener: "pages" as const, stage: "pick", input: "pages" },
        { opener: "keywords" as const, stage: "pick", input: "keywords" },
        { opener: "seed" as const, stage: "review", input: "pages" },
    ])("starts the $opener opener blank, on $stage", ({ opener, stage, input }) => {
        const held = memoryOf("s1", opener);
        expect(held).toBe(blankMemory(opener));
        expect(held.stage).toBe(stage);
        expect(held.pick.input).toBe(input);
        expect(held.pick.selected.size).toBe(0);
        expect(held.proposals).toHaveLength(0);
        expect(held.edits.size).toBe(0);
        expect(held.query).toBe("");
        expect(held.filter).toBe("all");
        expect(held.collapsed.size).toBe(0);
        expect(held.focus).toBeNull();
        expect(held.anchor).toBeNull();
        expect(held.pageTypes.size).toBe(0);
        expect(held.typeChoice).toBeNull();
        expect(held.seed).toBeNull();
    });

    it("keeps the source pages' types, the chosen type and the range's anchor with the review", () => {
        const pageTypes = new Map([["p1", "t-cat"]]);
        remember("s1", "seed", { pageTypes, typeChoice: "t-cat", anchor: 2 });
        remember("s1", "seed", { focus: 1 });

        const held = memoryOf("s1", "seed");
        expect(held.pageTypes).toBe(pageTypes);
        expect(held.typeChoice).toBe("t-cat");
        expect(held.anchor).toBe(2);
        expect(memoryOf("s2", "seed").typeChoice).toBeNull();
    });

    it("keeps what a site's opener built apart from every other site and opener", () => {
        const selected = new Set(["mugs", "travel"]);
        remember("s1", "pages", { pick: { ...blankMemory("pages").pick, selected } });

        expect(memoryOf("s1", "pages").pick.selected).toBe(selected);
        expect(memoryOf("s2", "pages")).toBe(blankMemory("pages"));
        expect(memoryOf("s1", "keywords")).toBe(blankMemory("keywords"));
        expect(memoryOf("s1", "seed")).toBe(blankMemory("seed"));
    });

    it("merges a patch over what it holds and hands back the same value until it changes", () => {
        remember("s1", "keywords", { stage: "review" });
        remember("s1", "keywords", { kept: new Set([0]) });

        const held = memoryOf("s1", "keywords");
        expect(held.stage).toBe("review");
        expect([...held.kept]).toStrictEqual([0]);
        expect(memoryOf("s1", "keywords")).toBe(held);
    });

    it("forgets one site's opener back to blank", () => {
        remember("s1", "pages", { stage: "confirm" });
        remember("s2", "pages", { stage: "confirm" });

        forgetProposal("s1", "pages");

        expect(memoryOf("s1", "pages")).toBe(blankMemory("pages"));
        expect(memoryOf("s2", "pages").stage).toBe("confirm");
    });
});

describe("seedKey", () => {
    it.each([
        { a: { siteId: "s1", pageIds: ["a", "b"] }, b: { siteId: "s1", pageIds: ["b", "a"] }, same: true },
        { a: { siteId: "s1", pageIds: ["a"] }, b: { siteId: "s1", pageIds: ["a", "b"] }, same: false },
        { a: { siteId: "s1", pageIds: ["a"] }, b: { siteId: "s2", pageIds: ["a"] }, same: false },
    ])("names the same findings seed alike whatever its order: $same", ({ a, b, same }) => {
        expect(seedKey(a) === seedKey(b)).toBe(same);
    });
});
