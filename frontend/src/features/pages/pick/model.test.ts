import { describe, expect, it } from "vitest";

import type { Page, PageTreeNode } from "../../../data/types.js";
import { branchOf, everyPage, indexTree, narrowTree, tick, toggled } from "./model.js";

function aPage(id: string, path: string, overrides: Partial<Page> = {}): Page {
    return {
        id,
        siteId: "s1",
        path,
        slug: "",
        parentPageId: null,
        wpType: "page",
        wpId: null,
        title: `Title ${id}`,
        h1: "",
        metaTitle: "",
        metaDescription: "",
        canonical: "",
        primaryKeyword: "",
        keywords: [],
        status: "planned",
        entityId: null,
        templateId: null,
        contentHash: "",
        observed: { link: "", slug: "", status: "", title: "", h1: "" },
        mismatches: [],
        wpModifiedAt: null,
        lastSyncedAt: null,
        drift: false,
        createdAt: "2026-09-24T09:00:00Z",
        updatedAt: "2026-09-24T09:00:00Z",
        ...overrides,
    };
}

const roots: PageTreeNode[] = [
    {
        page: aPage("shop", "/shop/", { entityId: "e1" }),
        children: [
            { page: aPage("mugs", "/shop/mugs/"), children: [{ page: aPage("travel", "/shop/mugs/travel/", { title: "Travel mugs" }), children: [] }] },
            { page: aPage("bags", "/shop/bags/", { entityId: "e2" }), children: [] },
        ],
    },
    { page: aPage("about", "/about/"), children: [] },
];

const unmapped = (page: Page): boolean => page.entityId === null;

describe("narrowTree", () => {
    it("keeps a match with the parents that lead to it", () => {
        const kept = narrowTree(roots, "travel", everyPage);
        expect(kept.map((node) => node.page.id)).toStrictEqual(["shop"]);
        expect(kept[0]?.children?.map((node) => node.page.id)).toStrictEqual(["mugs"]);
    });

    it("keeps only the pages a predicate accepts, with their ancestors", () => {
        const kept = narrowTree(roots, "", unmapped);
        expect(kept.map((node) => node.page.id)).toStrictEqual(["shop", "about"]);
        expect(kept[0]?.children?.map((node) => node.page.id)).toStrictEqual(["mugs"]);
    });

    it("answers the whole tree when nothing narrows it", () => {
        expect(narrowTree(roots, " ", everyPage)).toBe(roots);
        expect(narrowTree(null, "x", everyPage)).toStrictEqual([]);
    });
});

describe("branchOf", () => {
    it("takes a page and everything below it that the predicate accepts", () => {
        const index = indexTree(roots);
        expect(branchOf("shop", index, unmapped)).toStrictEqual(["mugs", "travel", "about"].filter((id) => id !== "about"));
        expect(branchOf("shop", index, everyPage)).toStrictEqual(["shop", "mugs", "bags", "travel"]);
    });
});

describe("tick and toggled", () => {
    it("reads chosen, required, partly chosen and untouched", () => {
        const index = indexTree(roots);
        const selected = new Set(["travel"]);
        const required = new Map([["mugs", "/shop/mugs/travel/"]]);
        expect(tick("travel", selected, required, index)).toBe("on");
        expect(tick("mugs", selected, required, index)).toBe("on");
        expect(tick("shop", selected, new Map(), index)).toBe("some");
        expect(tick("about", selected, required, index)).toBe("off");
    });

    it("adds and removes ids without touching the set it was given", () => {
        const held: ReadonlySet<string> = new Set(["a"]);
        const more = toggled(held, ["b", "c"], true);
        expect([...more]).toStrictEqual(["a", "b", "c"]);
        expect([...toggled(more, ["a", "c"], false)]).toStrictEqual(["b"]);
        expect([...held]).toStrictEqual(["a"]);
    });
});
