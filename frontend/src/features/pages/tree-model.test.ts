import { describe, expect, it } from "vitest";

import type { Page, PageTreeNode } from "../../data/types.js";
import { branchIds, countNodes, flattenTree, pathTo } from "./tree-model.js";

function page(id: string, path: string): Page {
    return {
        id,
        siteId: "site",
        path,
        slug: id,
        parentPageId: null,
        wpType: "page",
        wpId: null,
        title: id,
        h1: id,
        metaTitle: "",
        metaDescription: "",
        canonical: "",
        status: "planned",
        entityId: null,
        templateId: null,
        contentHash: "",
        observed: { link: "", slug: "", status: "", title: "", h1: "" },
        mismatches: [],
        wpModifiedAt: null,
        lastSyncedAt: null,
        drift: false,
        createdAt: "2026-09-19T10:00:00Z",
        updatedAt: "2026-09-19T10:00:00Z",
    };
}

function node(id: string, children: PageTreeNode[] | null = null): PageTreeNode {
    return { page: page(id, `/${id}/`), children };
}

const sample: PageTreeNode[] = [
    node("a", [node("a1", [node("a1x")]), node("a2")]),
    node("b"),
];

const none = new Set<string>();

describe("countNodes", () => {
    it("counts every node in the tree", () => {
        expect(countNodes(sample)).toBe(5);
    });

    it("treats a null response as an empty tree", () => {
        expect(countNodes(null)).toBe(0);
        expect(countNodes(undefined)).toBe(0);
    });
});

describe("branchIds", () => {
    it("lists only the nodes that have children", () => {
        expect([...branchIds(sample)].sort()).toStrictEqual(["a", "a1"]);
    });
});

describe("pathTo", () => {
    it("returns the ancestors of a node, deepest last", () => {
        expect(pathTo(sample, "a1x")).toStrictEqual(["a", "a1"]);
    });

    it("returns nothing for a root or an unknown id", () => {
        expect(pathTo(sample, "b")).toStrictEqual([]);
        expect(pathTo(sample, "missing")).toStrictEqual([]);
    });
});

describe("flattenTree", () => {
    it("renders only the roots while nothing is expanded", () => {
        const rows = flattenTree(sample, none, none, 200);
        expect(rows.map((row) => row.id)).toStrictEqual(["a", "b"]);
    });

    it("renders the children of an expanded node and nothing below a collapsed one", () => {
        const rows = flattenTree(sample, new Set(["a"]), none, 200);
        expect(rows.map((row) => row.id)).toStrictEqual(["a", "a1", "a2", "b"]);
    });

    it("descends every expanded level", () => {
        const rows = flattenTree(sample, new Set(["a", "a1"]), none, 200);
        expect(rows.map((row) => row.id)).toStrictEqual(["a", "a1", "a1x", "a2", "b"]);
    });

    it("reports a child count and an expansion flag on every branch", () => {
        const rows = flattenTree(sample, new Set(["a"]), none, 200);
        const first = rows[0];
        expect(first.kind === "page" && first.childCount).toBe(2);
        expect(first.kind === "page" && first.expanded).toBe(true);
        const leaf = rows[2];
        expect(leaf.kind === "page" && leaf.childCount).toBe(0);
        expect(leaf.kind === "page" && leaf.expanded).toBe(false);
    });

    it("caps the siblings it emits and reports how many it withheld", () => {
        const wide: PageTreeNode[] = [node("root", Array.from({ length: 10 }, (_unused, i) => node(`c${i}`)))];
        const rows = flattenTree(wide, new Set(["root"]), none, 4);
        expect(rows.map((row) => row.id)).toStrictEqual([
            "root",
            "c0",
            "c1",
            "c2",
            "c3",
            "more:root",
        ]);
        const marker = rows[5];
        expect(marker.kind === "more" && marker.hidden).toBe(6);
    });

    it("emits every sibling once the cap is lifted for that parent", () => {
        const wide: PageTreeNode[] = [node("root", Array.from({ length: 10 }, (_unused, i) => node(`c${i}`)))];
        const rows = flattenTree(wide, new Set(["root"]), new Set(["root"]), 4);
        expect(rows).toHaveLength(11);
        expect(rows.every((row) => row.kind === "page")).toBe(true);
    });

    it("caps the root level too", () => {
        const wide = Array.from({ length: 6 }, (_unused, i) => node(`r${i}`));
        const rows = flattenTree(wide, none, none, 2);
        expect(rows.map((row) => row.id)).toStrictEqual(["r0", "r1", "more:"]);
    });

    it("treats a null response as an empty tree", () => {
        expect(flattenTree(null, none, none, 200)).toStrictEqual([]);
    });
});
