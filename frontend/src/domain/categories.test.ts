import { describe, expect, it } from "vitest";

import { copy } from "../copy/index.js";
import type { Category } from "./categories.js";
import {
    becomesCategory,
    categoryStanding,
    entityCategoryItems,
    filedItems,
    ownCategoryItem,
    pageCategoryItems,
} from "./categories.js";

const peptides: Category = { entityId: "peptides", name: "Peptides", termId: 12 };
const healing: Category = { entityId: "healing", name: "Healing" };

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
            [{ entityId: "zero", name: "Zero", termId: 0 }],
            false,
            [{ name: "Zero", state: "onPublish", hint: copy.categories.onPublish }],
        ],
    ])("maps %s", (_, categories, categoriesNeedPlugin, want) => {
        const items = pageCategoryItems({ categories, categoriesNeedPlugin });
        expect(items.map(({ name, state, hint }) => ({ name, state, hint }))).toStrictEqual(want);
    });

    it("keys each chip by the entity it stands for", () => {
        expect(pageCategoryItems({ categories: [peptides, healing], categoriesNeedPlugin: false }).map((item) => item.key)).toStrictEqual([
            "peptides",
            "healing",
        ]);
    });
});

describe("entityCategoryItems", () => {
    it("never warns about the plugin, which only a page needs", () => {
        const items = entityCategoryItems({ categories: [peptides, healing] });
        expect(items.map((item) => item.state)).toStrictEqual(["onSite", "onPublish"]);
    });

    it("maps an entity under no category to nothing", () => {
        expect(entityCategoryItems({ categories: null })).toStrictEqual([]);
    });
});

describe("filedItems", () => {
    it("says which terms the run created and which it found", () => {
        const items = filedItems([
            { entityId: "peptides", name: "Peptides", termId: 12, created: false },
            { entityId: "healing", name: "Healing", termId: 31, created: true },
        ]);
        expect(items).toStrictEqual([
            { key: "peptides", name: "Peptides", state: "onSite", hint: copy.categories.onSite(12) },
            { key: "healing", name: "Healing", state: "onSite", hint: copy.categories.createdByRun(31) },
        ]);
    });
});

describe("becomesCategory", () => {
    it("is the plain chip an import preview shows", () => {
        expect(becomesCategory()).toStrictEqual({
            key: "becomes",
            name: copy.categories.becomes,
            state: "becomes",
            hint: copy.categories.becomesHint,
        });
    });
});

describe("categoryStanding", () => {
    it.each<[string, { id: string; siteCategory: boolean; categories: Category[] | null }, ReturnType<typeof categoryStanding>]>([
        ["an entity that is not a category", { id: "healing", siteCategory: false, categories: [peptides] }, { state: "off" }],
        [
            "a category the site already has",
            { id: "peptides", siteCategory: true, categories: [peptides] },
            { state: "onSite", termId: 12 },
        ],
        [
            "a category not created yet",
            { id: "healing", siteCategory: true, categories: [peptides, healing] },
            { state: "onPublish" },
        ],
        ["a category whose chain has not loaded", { id: "healing", siteCategory: true, categories: null }, { state: "onPublish" }],
    ])("reads %s", (_, entity, want) => {
        expect(categoryStanding(entity)).toStrictEqual(want);
    });
});

describe("ownCategoryItem", () => {
    it.each<[string, { id: string; siteCategory: boolean; categories: Category[] | null }, ReturnType<typeof ownCategoryItem>]>([
        ["nothing for an entity that is not a category", { id: "healing", siteCategory: false, categories: [] }, null],
        [
            "the term of a category on the site",
            { id: "peptides", siteCategory: true, categories: [peptides] },
            { key: "peptides", name: "Marker", state: "onSite", hint: copy.categories.onSite(12) },
        ],
        [
            "the promise of a category not created yet",
            { id: "healing", siteCategory: true, categories: [peptides, healing] },
            { key: "healing", name: "Marker", state: "onPublish", hint: copy.categories.onPublish },
        ],
    ])("gives %s", (_, entity, want) => {
        expect(ownCategoryItem(entity, "Marker")).toStrictEqual(want);
    });
});
