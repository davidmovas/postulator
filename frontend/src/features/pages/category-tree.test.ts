import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import type { CategoryNode } from "../../data/types.js";
import { ancestorsOf, categoryChip, categoryRows, categoryTree } from "./category-tree.js";

function node(id: string, name: string, parentId: string | null, part: Partial<CategoryNode> = {}): CategoryNode {
    return { id, name, parentId, pages: 1, termIds: {}, ...part };
}

const listed: CategoryNode[] = [
    node("peptides", "Peptides", null, { pages: 5 }),
    node("tools", "Tools", null, { pages: 1 }),
    node("healing", "Healing", "peptides", { pages: 3 }),
    node("growth", "Growth", "peptides", { pages: 2 }),
    node("liquid", "Liquid", "healing", { pages: 3 }),
];

function shown(open: readonly string[]): [string, number, boolean, boolean][] {
    return categoryRows(categoryTree(listed), new Set(open)).map((row) => [row.node.id, row.depth, row.branches, row.open]);
}

describe("the category tree of the rail", () => {
    it.each<[string, string[], [string, number, boolean, boolean][]]>([
        [
            "only the roots while nothing is open",
            [],
            [
                ["peptides", 0, true, false],
                ["tools", 0, false, false],
            ],
        ],
        [
            "the children of an open root, in the order they came",
            ["peptides"],
            [
                ["peptides", 0, true, true],
                ["healing", 1, true, false],
                ["growth", 1, false, false],
                ["tools", 0, false, false],
            ],
        ],
        [
            "a whole branch open to its leaf",
            ["peptides", "healing"],
            [
                ["peptides", 0, true, true],
                ["healing", 1, true, true],
                ["liquid", 2, false, false],
                ["growth", 1, false, false],
                ["tools", 0, false, false],
            ],
        ],
        [
            "nothing under a branch whose parent is shut",
            ["healing"],
            [
                ["peptides", 0, true, false],
                ["tools", 0, false, false],
            ],
        ],
        ["a leaf that cannot open", ["tools"], [["peptides", 0, true, false], ["tools", 0, false, false]]],
    ])("shows %s", (_, open, want) => {
        expect(shown(open)).toStrictEqual(want);
    });

    it("hangs a category whose parent is missing at the root rather than losing it", () => {
        const rows = categoryRows(categoryTree([node("orphan", "Orphan", "gone")]), new Set());
        expect(rows.map((row) => row.node.id)).toStrictEqual(["orphan"]);
    });

    it("names the categories above one, root first, so the rail can open them", () => {
        const tree = categoryTree(listed);
        expect(ancestorsOf(tree, "liquid")).toStrictEqual(["peptides", "healing"]);
        expect(ancestorsOf(tree, "peptides")).toStrictEqual([]);
        expect(ancestorsOf(tree, "unknown")).toStrictEqual([]);
    });

    it("stops at a loop instead of walking it forever", () => {
        const tree = categoryTree([node("a", "A", "b"), node("b", "B", "a"), node("c", "C", "a")]);
        expect(ancestorsOf(tree, "c")).toStrictEqual(["b", "a"]);
    });
});

describe("the state chip of a category", () => {
    const said = copy.pages.categories;

    it.each<[string, CategoryNode["termIds"], { state: string; label: string; hint: string }]>([
        ["a category WordPress holds", { category: 12 }, { state: "onSite", label: "#12", hint: said.onSite(12) }],
        [
            "a category held as a page and a product category",
            { category: 12, productCategory: 31 },
            { state: "onSite", label: "#12", hint: said.onSiteBoth(12, 31) },
        ],
        [
            "a category the store holds for its products only",
            { productCategory: 31 },
            { state: "onSite", label: "#31", hint: said.onSiteProduct(31) },
        ],
        ["a category not on the site yet", {}, { state: "onPublish", label: said.newLabel, hint: said.onPublish }],
        [
            "term ids that no WordPress term carries",
            { category: 0, productCategory: null },
            { state: "onPublish", label: said.newLabel, hint: said.onPublish },
        ],
    ])("reads %s", (_, termIds, want) => {
        expect(categoryChip(node("c", "C", null, { termIds }))).toStrictEqual(want);
    });
});
