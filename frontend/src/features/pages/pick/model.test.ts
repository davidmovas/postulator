import { describe, expect, it } from "vitest";

import type { SiteMapNode } from "../../../data/types.js";
import { aNode } from "../../map/model/fixture.js";
import type { Branches } from "../../map/model/tree.js";
import { everyNode, kept, labelOf, tick, toggled } from "./model.js";

const shop = aNode("shop", "/shop/", { children: 2, descendants: 3 });
const mugs = aNode("mugs", "/shop/mugs/", { children: 1, descendants: 1 });
const bags = aNode("bags", "/shop/bags/");
const travel = aNode("travel", "/shop/mugs/travel/");

const branches: Branches = new Map([
    ["", { nodes: [shop], hasMore: false, pending: false }],
    ["shop", { nodes: [mugs, bags], hasMore: false, pending: false }],
    ["mugs", { nodes: [travel], hasMore: false, pending: false }],
]);

const onSite = (node: SiteMapNode): boolean => node.page.onSite;

describe("tick", () => {
    it("reads picked, required, partly picked below and untouched over the loaded branches", () => {
        const selected = new Set(["travel"]);
        const required = new Map([["mugs", "/shop/mugs/travel/"]]);
        expect(tick("travel", selected, required, branches)).toBe("on");
        expect(tick("mugs", selected, required, branches)).toBe("on");
        expect(tick("shop", selected, required, branches)).toBe("some");
        expect(tick("bags", selected, required, branches)).toBe("off");
    });

    it("reads a page picked from a branch that is not loaded as picked", () => {
        expect(tick("elsewhere", new Set(["elsewhere"]), new Map(), branches)).toBe("on");
    });
});

describe("toggled", () => {
    it("adds or drops the ids and keeps every other pick", () => {
        expect([...toggled(new Set(["a"]), ["b", "c"], true)]).toStrictEqual(["a", "b", "c"]);
        expect([...toggled(new Set(["a", "b"]), ["a"], false)]).toStrictEqual(["b"]);
    });
});

describe("kept", () => {
    it("keeps a node the filter accepts and a node with children to open", () => {
        const missing = (node: SiteMapNode): boolean => !onSite(node);
        expect(kept(aNode("x", "/x/", { page: { ...shop.page, onSite: false } }), missing)).toBe(true);
        expect(kept(bags, missing)).toBe(false);
        expect(kept(shop, missing)).toBe(true);
        expect(kept(bags, everyNode)).toBe(true);
    });
});

describe("labelOf", () => {
    it("names a tree row by its last segment and a search match by its whole path", () => {
        expect(labelOf(travel, false)).toBe("travel/");
        expect(labelOf(travel, true)).toBe("/shop/mugs/travel/");
        expect(labelOf(aNode("root", "/"), false)).toBe("/");
    });
});
