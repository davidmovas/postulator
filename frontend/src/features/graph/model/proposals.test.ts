import { describe, expect, it } from "vitest";

import type { ProposedEntity } from "../../../data/types.js";
import { aPage } from "../../../testing/pages.js";
import { chosenOf, everyTickable, keywordLines, placeOf, tickable, toggled, unmapped, writesOf } from "./proposals.js";

const mugs: ProposedEntity = { pageId: "p1", path: "/shop/mugs/", name: "Mugs", label: "Mugs", parent: "/shop/" };
const cups: ProposedEntity = { pageId: "p2", path: "/shop/cups/", name: "Cups", parent: "/shop/", existingEntityId: "e9" };
const twin: ProposedEntity = { pageId: "p3", path: "/shop/mugs-2/", name: "Mugs", parent: "/shop/", clash: "name_in_batch" };
const nameless: ProposedEntity = { pageId: "p4", path: "/shop/x/", name: "", clash: "no_name" };

describe("keywordLines", () => {
    it("takes one keyword per line, trimmed and without repeats", () => {
        expect(keywordLines(" trail shoes \n\nroad shoes\r\nTrail Shoes\n")).toStrictEqual(["trail shoes", "road shoes"]);
    });
});

describe("unmapped", () => {
    it("accepts a page without an entity and never the root", () => {
        expect(unmapped(aPage("a", "/a/"))).toBe(true);
        expect(unmapped(aPage("a", "/a/", { entityId: "e1" }))).toBe(false);
        expect(unmapped(aPage("root", "/"))).toBe(false);
    });
});

describe("placeOf", () => {
    it("names the page a proposal carries, else the parent it sits under", () => {
        expect(placeOf(mugs)).toBe("/shop/mugs/");
        expect(placeOf({ name: "Trail shoes", parent: "/shoes/" })).toBe("/shoes/");
        expect(placeOf({ name: "Trail shoes" })).toBe("");
    });
});

describe("the proposals a person can tick", () => {
    it("refuses a proposal that clashes and takes every other", () => {
        expect(tickable(mugs)).toBe(true);
        expect(tickable(cups)).toBe(true);
        expect(tickable(twin)).toBe(false);
        expect(tickable(nameless)).toBe(false);
    });

    it("ticks every proposal but the clashing ones", () => {
        expect([...everyTickable([mugs, twin, cups, nameless])]).toStrictEqual([0, 2]);
    });

    it("flips one row and never a clashing one", () => {
        const proposals = [mugs, twin];
        expect([...toggled(proposals, new Set([0]), 0)]).toStrictEqual([]);
        expect([...toggled(proposals, new Set<number>(), 0)]).toStrictEqual([0]);
        expect([...toggled(proposals, new Set([0]), 1)]).toStrictEqual([0]);
    });

    it("sends only the ticked rows that do not clash", () => {
        expect(chosenOf([mugs, twin, cups], new Set([0, 1]))).toStrictEqual([mugs]);
    });

    it("counts the entities written apart from the pages joining an entity that exists", () => {
        expect(writesOf([mugs, cups])).toStrictEqual({ written: 1, joined: 1 });
        expect(writesOf([])).toStrictEqual({ written: 0, joined: 0 });
    });
});
