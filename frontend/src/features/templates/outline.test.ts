import { describe, expect, it } from "vitest";

import type { JsonObject } from "../../domain/merge-patch.js";
import type { GroupKey } from "./outline.js";
import { changedGroups, groupKeys, groupTitles, isGroup } from "./outline.js";

describe("changedGroups", () => {
    const cases: readonly { name: string; patch: JsonObject | null; want: GroupKey[] }[] = [
        { name: "no patch changes nothing", patch: null, want: [] },
        { name: "an empty patch changes nothing", patch: {}, want: [] },
        { name: "a section rewrite marks sections", patch: { sections: [] }, want: ["sections", "overrides"] },
        {
            name: "a length change marks rules",
            patch: { length: { max: 900 } },
            want: ["rules", "overrides"],
        },
        {
            name: "a link rule marks rules",
            patch: { linkRules: { maxLinks: 4 } },
            want: ["rules", "overrides"],
        },
        { name: "the tone marks rules", patch: { tone: "plain" }, want: ["rules", "overrides"] },
        {
            name: "a pinned model marks models",
            patch: { modelProfiles: { writer: { provider: "openai", model: "gpt-5.1" } } },
            want: ["models", "overrides"],
        },
        { name: "a recipe marks the recipe", patch: { recipe: [] }, want: ["recipe", "overrides"] },
        {
            name: "several groups at once",
            patch: { sections: [], images: { inline: 2 }, recipe: [] },
            want: ["sections", "rules", "recipe", "overrides"],
        },
        {
            name: "a key no group watches still counts as an override",
            patch: { somethingElse: true },
            want: [],
        },
    ];

    for (const held of cases) {
        it(held.name, () => {
            expect([...changedGroups(held.patch)].sort()).toEqual([...held.want].sort());
        });
    }

    it("never marks the policies group, which this layer cannot change", () => {
        expect(changedGroups({ sections: [], recipe: [] }).has("policies")).toBe(false);
    });
});

describe("groupKeys", () => {
    it("names every group it lists", () => {
        for (const key of groupKeys) {
            expect(groupTitles[key]).not.toBe("");
            expect(isGroup(key)).toBe(true);
        }
        expect(isGroup("nonsense")).toBe(false);
    });
});
