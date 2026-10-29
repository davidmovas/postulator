import { describe, expect, it } from "vitest";

import type { SiteMapNode } from "../../../data/types.js";
import { aNode } from "../../map/model/fixture.js";
import type { Branches } from "../../map/model/tree.js";
import type { NamedPage } from "./model.js";
import { everyNode, kept, labelOf, nextPick, tick, toggled } from "./model.js";

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

describe("nextPick", () => {
    const page = (id: string): NamedPage => ({ id, path: `/${id}/` });
    const shown = ["a", "b", "c", "d"].map(page);

    const cases: {
        name: string;
        selected: readonly string[];
        anchor: string | null;
        target: string;
        ranged: boolean;
        want: readonly string[];
        named: readonly string[];
        anchored: string;
    }[] = [
        {
            name: "a plain click toggles the page and anchors on it",
            selected: ["a"],
            anchor: "a",
            target: "c",
            ranged: false,
            want: ["a", "c"],
            named: ["c"],
            anchored: "c",
        },
        {
            name: "a shift-click sets the shown pages between to the anchor's state and keeps the anchor",
            selected: ["a"],
            anchor: "a",
            target: "d",
            ranged: true,
            want: ["a", "b", "c", "d"],
            named: ["a", "b", "c", "d"],
            anchored: "a",
        },
        {
            name: "a shift-click upwards from an unticked anchor unticks the span",
            selected: ["a", "b"],
            anchor: "c",
            target: "a",
            ranged: true,
            want: [],
            named: ["a", "b", "c"],
            anchored: "c",
        },
        {
            name: "a shift-click with no anchor toggles the page alone and anchors on it",
            selected: [],
            anchor: null,
            target: "b",
            ranged: true,
            want: ["b"],
            named: ["b"],
            anchored: "b",
        },
        {
            name: "a shift-click whose anchor is not shown toggles the page alone",
            selected: ["gone"],
            anchor: "gone",
            target: "b",
            ranged: true,
            want: ["b", "gone"],
            named: ["b"],
            anchored: "b",
        },
        {
            name: "a page outside the shown rows keeps its pick",
            selected: ["kept", "a"],
            anchor: "a",
            target: "b",
            ranged: true,
            want: ["a", "b", "kept"],
            named: ["a", "b"],
            anchored: "a",
        },
    ];

    it.each(cases)("$name", ({ selected, anchor, target, ranged, want, named, anchored }) => {
        const next = nextPick(new Set(selected), shown, anchor, page(target), ranged);
        expect([...next.selected].sort()).toStrictEqual([...want].sort());
        expect(next.named.map((held) => held.id)).toStrictEqual(named);
        expect(next.anchor).toBe(anchored);
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
