import { describe, expect, it } from "vitest";

import { aNode } from "./fixture.js";
import type { Branch, Branches } from "./tree.js";
import { flattenTree, openBranches, toggled } from "./tree.js";

const root = "";

const shop = aNode("shop", "/shop/", { children: 2 });
const lab = aNode("lab", "/lab/", { children: 1 });
const about = aNode("about", "/about/");
const mugs = aNode("mugs", "/shop/mugs/", { children: 3 });
const cups = aNode("cups", "/shop/cups/");
const guides = aNode("guides", "/lab/guides/");

function branches(entries: readonly (readonly [string, Partial<Branch>])[]): Branches {
    return new Map(entries.map(([key, value]) => [key, { nodes: [], hasMore: false, pending: false, ...value }]));
}

describe("which branches of the tree ask the site for their children", () => {
    it("asks for the roots alone while nothing is expanded", () => {
        expect(openBranches(root, new Set(), branches([[root, { nodes: [shop, lab, about] }]]))).toStrictEqual([root]);
    });

    it("asks for an expanded node's children, and for an expanded child under it", () => {
        const loaded = branches([
            [root, { nodes: [shop, lab, about] }],
            ["shop", { nodes: [mugs, cups] }],
        ]);
        expect(openBranches(root, new Set(["shop", "mugs"]), loaded)).toStrictEqual([root, "shop", "mugs"]);
    });

    it("stops asking under a collapsed ancestor, though the expanded mark is kept", () => {
        const loaded = branches([
            [root, { nodes: [shop, lab, about] }],
            ["shop", { nodes: [mugs, cups] }],
        ]);
        expect(openBranches(root, new Set(["mugs"]), loaded)).toStrictEqual([root]);
    });

    it("never asks for a node with no children, whatever is marked", () => {
        expect(openBranches(root, new Set(["about"]), branches([[root, { nodes: [about] }]]))).toStrictEqual([root]);
    });

    it("roots the tree at a subtree", () => {
        expect(openBranches("shop", new Set(["mugs"]), branches([["shop", { nodes: [mugs, cups] }]]))).toStrictEqual(["shop", "mugs"]);
    });
});

describe("the rows the tree renders", () => {
    it("says the roots are loading before they arrive", () => {
        expect(flattenTree(root, new Set(), new Map())).toStrictEqual([{ kind: "loading", key: "loading:", parentId: root, depth: 0 }]);
    });

    it("lists the roots, then an expanded node's children one level deeper", () => {
        const loaded = branches([
            [root, { nodes: [shop, lab] }],
            ["shop", { nodes: [mugs, cups] }],
        ]);
        const rows = flattenTree(root, new Set(["shop"]), loaded);
        expect(rows.map((row) => (row.kind === "node" ? `${row.depth}:${row.node.page.pageId}:${row.expanded}` : `${row.depth}:${row.kind}`))).toStrictEqual([
            "0:shop:true",
            "1:mugs:false",
            "1:cups:false",
            "0:lab:false",
        ]);
    });

    it("shows an expanded node's children loading, and a more row where a branch has another page", () => {
        const loaded = branches([
            [root, { nodes: [shop, lab], hasMore: true }],
            ["shop", { nodes: [mugs], hasMore: true }],
            ["lab", { pending: true }],
        ]);
        const rows = flattenTree(root, new Set(["shop", "lab", "mugs"]), loaded);
        expect(rows.map((row) => `${row.depth}:${row.kind}:${row.kind === "node" ? row.node.page.pageId : row.parentId}`)).toStrictEqual([
            "0:node:shop",
            "1:node:mugs",
            "2:loading:mugs",
            "1:more:shop",
            "0:node:lab",
            "1:loading:lab",
            "0:more:",
        ]);
    });

    it("keeps every row's key unique", () => {
        const loaded = branches([
            [root, { nodes: [shop, lab], hasMore: true }],
            ["shop", { nodes: [mugs, cups], hasMore: true }],
            ["lab", { nodes: [guides] }],
        ]);
        const keys = flattenTree(root, new Set(["shop", "lab"]), loaded).map((row) => row.key);
        expect(new Set(keys).size).toBe(keys.length);
    });
});

describe("expanding a node", () => {
    it("adds a node and takes it away again", () => {
        const opened = toggled(new Set(), "shop");
        expect([...opened]).toStrictEqual(["shop"]);
        expect([...toggled(opened, "shop")]).toStrictEqual([]);
    });
});
