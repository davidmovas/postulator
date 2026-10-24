import { describe, expect, it } from "vitest";

import { defaultQuery, filterOf, flat, readQuery, searchOf, writeQuery } from "./params.js";

describe("the site map's filters in the address", () => {
    it("shows the whole tree by default", () => {
        expect(defaultQuery).toStrictEqual({ under: "", search: "", type: "" });
        expect(searchOf(defaultQuery)).toBe("");
        expect(flat(defaultQuery)).toBe(false);
    });

    it("round-trips a subtree, a search and a type", () => {
        const query = { under: "p1", search: "mug", type: "pt-hub" };
        expect(writeQuery(query).toString()).toBe("under=p1&q=mug&type=pt-hub");
        expect(readQuery(writeQuery(query))).toStrictEqual(query);
    });

    it("lists flat results while a search or a type is set", () => {
        expect(flat({ ...defaultQuery, search: "mug" })).toBe(true);
        expect(flat({ ...defaultQuery, type: "pt-hub" })).toBe(true);
        expect(flat({ ...defaultQuery, under: "p1" })).toBe(false);
    });
});

describe("what the site map asks the site for", () => {
    it("asks for the roots, or the children of the subtree's root", () => {
        expect(filterOf("s1", defaultQuery)).toStrictEqual({ siteId: "s1" });
        expect(filterOf("s1", { ...defaultQuery, under: "p1" })).toStrictEqual({ siteId: "s1", parentId: "p1" });
    });

    it("asks for the matches of a trimmed search and a type, under the subtree when one is set", () => {
        expect(filterOf("s1", { under: "p1", search: "  mug ", type: "pt-hub" })).toStrictEqual({
            siteId: "s1",
            parentId: "p1",
            search: "mug",
            pageTypeId: "pt-hub",
        });
    });
});
