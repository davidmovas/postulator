import { describe, expect, it } from "vitest";

import { copy } from "../copy/index.js";
import type { Category } from "./categories.js";
import { chainItems, filedItems, pageCategoryItems } from "./categories.js";

const peptides: Category = { id: "peptides", name: "Peptides", termId: 12 };
const healing: Category = { id: "healing", name: "Healing" };

describe("pageCategoryItems", () => {
    it.each<[string, Category[] | null, boolean, { name: string; state: string; hint: string }[]]>([
        [
            "a chain whose root is on the site and whose leaf is not yet",
            [peptides, healing],
            false,
            [
                { name: "Peptides", state: "onSite", hint: copy.categories.onSite(12) },
                { name: "Healing", state: "onPublish", hint: copy.categories.onPublish },
            ],
        ],
        [
            "a chain a page cannot carry until the plugin is updated",
            [peptides, healing],
            true,
            [
                { name: "Peptides", state: "needsPlugin", hint: copy.categories.needsPlugin },
                { name: "Healing", state: "needsPlugin", hint: copy.categories.needsPlugin },
            ],
        ],
        ["a page no category files", [], false, []],
        ["a page whose categories arrived as null", null, false, []],
        [
            "a term id of zero, which no WordPress term has",
            [{ id: "zero", name: "Zero", termId: 0 }],
            false,
            [{ name: "Zero", state: "onPublish", hint: copy.categories.onPublish }],
        ],
    ])("maps %s", (_, categories, categoriesNeedPlugin, want) => {
        const items = pageCategoryItems({ categories, categoriesNeedPlugin });
        expect(items.map(({ name, state, hint }) => ({ name, state, hint }))).toStrictEqual(want);
    });

    it("keys each chip by the category record it stands for", () => {
        expect(pageCategoryItems({ categories: [peptides, healing], categoriesNeedPlugin: false }).map((item) => item.key)).toStrictEqual([
            "peptides",
            "healing",
        ]);
    });
});

describe("chainItems", () => {
    it("says which categories of an entity's page are on the site, and never warns about the plugin", () => {
        const items = chainItems([peptides, healing]);
        expect(items.map((item) => [item.key, item.state, item.hint])).toStrictEqual([
            ["peptides", "onSite", copy.categories.onSite(12)],
            ["healing", "onPublish", copy.categories.onPublish],
        ]);
    });

    it.each([[null], [undefined], [[]]])("maps an entity with no filed page to nothing (%j)", (chain) => {
        expect(chainItems(chain)).toStrictEqual([]);
    });
});

describe("filedItems", () => {
    it("says which terms the run created and which it found, keyed by the term", () => {
        const items = filedItems([
            { name: "Peptides", termId: 12, created: false },
            { name: "Healing", termId: 31, created: true },
        ]);
        expect(items).toStrictEqual([
            { key: "12", name: "Peptides", state: "onSite", hint: copy.categories.onSite(12) },
            { key: "31", name: "Healing", state: "onSite", hint: copy.categories.createdByRun(31) },
        ]);
    });
});
