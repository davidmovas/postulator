import { describe, expect, it } from "vitest";

import type { ProposedEntity } from "../../../data/types.js";
import { aNode } from "../../map/model/fixture.js";
import { joinsExisting, keywordLines, unmapped, writesOf } from "./proposals.js";

const mugs: ProposedEntity = { pageId: "p1", path: "/shop/mugs/", name: "Mugs", label: "Mugs", parent: "/shop/" };
const cups: ProposedEntity = { pageId: "p2", path: "/shop/cups/", name: "Cups", parent: "/shop/", existingEntityId: "e9" };

describe("keywordLines", () => {
    it("takes one keyword per line, trimmed and without repeats", () => {
        expect(keywordLines(" trail shoes \n\nroad shoes\r\nTrail Shoes\n")).toStrictEqual(["trail shoes", "road shoes"]);
    });
});

describe("unmapped", () => {
    it("accepts a page without an entity and never the root", () => {
        expect(unmapped(aNode("a", "/a/"))).toBe(true);
        expect(unmapped(aNode("a", "/a/", { entityId: "" }))).toBe(true);
        expect(unmapped(aNode("a", "/a/", { entityId: "e1" }))).toBe(false);
        expect(unmapped(aNode("root", "/"))).toBe(false);
    });
});

describe("what a write does", () => {
    it("knows a proposal that joins an entity which exists", () => {
        expect(joinsExisting(mugs)).toBe(false);
        expect(joinsExisting(cups)).toBe(true);
    });

    it("counts the entities written apart from the pages joining an entity that exists", () => {
        expect(writesOf([mugs, cups])).toStrictEqual({ written: 1, joined: 1 });
        expect(writesOf([])).toStrictEqual({ written: 0, joined: 0 });
    });
});
