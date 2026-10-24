import { describe, expect, it } from "vitest";

import type { SiteMapNode } from "../../data/types.js";
import { aNode, aRef } from "../map/model/fixture.js";
import { idsOf, inScope, pickable, requiredOf, selectionKey } from "./start-selection.js";

const live = aNode("bikes", "/bikes/", { entityId: "e1" });
const planned = aNode("cargo", "/bikes/cargo/", {
    entityId: "e2",
    status: "planned" as SiteMapNode["status"],
    page: aRef("cargo", "/bikes/cargo/", { onSite: false }),
});
const unmapped = aNode("mini", "/bikes/cargo/mini/");

describe("pickable", () => {
    it("takes a page an entity carries and refuses one nothing is mapped to", () => {
        expect(pickable(live)).toBe(true);
        expect(pickable(unmapped)).toBe(false);
        expect(pickable(aNode("blank", "/blank/", { entityId: "" }))).toBe(false);
    });
});

describe("inScope", () => {
    it("shows every page, only those not on the site, or only those that are", () => {
        expect(inScope("all")(live)).toBe(true);
        expect(inScope("missing")(live)).toBe(false);
        expect(inScope("missing")(planned)).toBe(true);
        expect(inScope("present")(live)).toBe(true);
        expect(inScope("present")(planned)).toBe(false);
    });
});

describe("requiredOf", () => {
    it("names each parent the run writes first, with the page that needs it", () => {
        const required = requiredOf([{ pageId: "cargo", path: "/bikes/cargo/", neededBy: "/bikes/cargo/max/" }], new Set(["max"]));
        expect([...required.entries()]).toStrictEqual([["cargo", "/bikes/cargo/max/"]]);
    });

    it("names no parent that is picked itself, and none for an empty pick", () => {
        const added = [{ pageId: "cargo", path: "/bikes/cargo/", neededBy: "/bikes/cargo/max/" }];
        expect(requiredOf(added, new Set(["max", "cargo"])).size).toBe(0);
        expect(requiredOf(added, new Set()).size).toBe(0);
        expect(requiredOf(null, new Set(["max"])).size).toBe(0);
    });
});

describe("the selection the parents are asked for", () => {
    it("is the same key whatever order the pages were picked in, and gives the ids back", () => {
        expect(selectionKey(new Set(["b", "a"]))).toBe(selectionKey(new Set(["a", "b"])));
        expect(idsOf(selectionKey(new Set(["b", "a"])))).toStrictEqual(["a", "b"]);
        expect(idsOf(selectionKey(new Set()))).toStrictEqual([]);
    });
});
