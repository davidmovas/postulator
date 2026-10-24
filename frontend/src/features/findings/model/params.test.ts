import { describe, expect, it } from "vitest";

import { countsOf, defaultQuery, filterOf, findingsAddress, narrowed, readQuery, searchOf, writeQuery } from "./params.js";

describe("the inbox's filters in the address", () => {
    it("lists the open findings of every code, type and page by default", () => {
        expect(defaultQuery).toStrictEqual({ status: "open", code: "", type: "", under: "" });
        expect(searchOf(defaultQuery)).toBe("");
        expect(narrowed(defaultQuery)).toBe(false);
    });

    it("round-trips every filter", () => {
        const query = { status: "dismissed", code: "row_ambiguous", type: "pt-hub", under: "p1" } as const;
        const written = writeQuery(query);
        expect(written.toString()).toBe("status=dismissed&code=row_ambiguous&type=pt-hub&under=p1");
        expect(readQuery(written)).toStrictEqual(query);
        expect(narrowed(query)).toBe(true);
    });

    it("falls back on a status or a code this build does not know", () => {
        expect(readQuery(new URLSearchParams("status=lost&code=teleported"))).toStrictEqual(defaultQuery);
    });

    it("keeps the status out of what counts as narrowed", () => {
        expect(narrowed({ ...defaultQuery, status: "resolved" })).toBe(false);
    });
});

describe("what the inbox asks the site for", () => {
    it("names only the filters a person set", () => {
        expect(filterOf("s1", defaultQuery)).toStrictEqual({ siteId: "s1", status: "open" });
        expect(countsOf("s1", defaultQuery)).toStrictEqual({ siteId: "s1", status: "open" });
    });

    it("lists a code, a type and a subtree, and counts every code under the same type and subtree", () => {
        const query = { status: "open", code: "parent_differs", type: "pt-hub", under: "p1" } as const;
        expect(filterOf("s1", query)).toStrictEqual({ siteId: "s1", status: "open", code: "parent_differs", pageTypeId: "pt-hub", under: "p1" });
        expect(countsOf("s1", query)).toStrictEqual({ siteId: "s1", status: "open", pageTypeId: "pt-hub", under: "p1" });
    });
});

describe("the address another screen links to", () => {
    it("opens the open findings under a page", () => {
        expect(findingsAddress("s1", "p1")).toBe("/s/s1/findings?under=p1");
    });
});
