import { describe, expect, it } from "vitest";

import { defaultQuery, narrowed, readQuery, searchOf, sortChoice, sorts, writeQuery } from "./params.js";
import type { TemplatesQuery } from "./params.js";

describe("readQuery", () => {
    const cases: readonly { name: string; raw: string; want: TemplatesQuery }[] = [
        { name: "an empty search is the default query", raw: "", want: defaultQuery },
        {
            name: "every facet round-trips",
            raw: "scope=site&kind=guide&q=kiln&sort=name:desc",
            want: { scope: "site", pageKind: "guide", search: "kiln", sort: { field: "name", desc: true } },
        },
        {
            name: "an unknown scope falls back to all",
            raw: "scope=workspace",
            want: { ...defaultQuery, scope: "all" },
        },
        {
            name: "an unknown sort field is dropped",
            raw: "sort=updatedAt:desc",
            want: defaultQuery,
        },
    ];

    for (const held of cases) {
        it(held.name, () => {
            expect(readQuery(new URLSearchParams(held.raw))).toEqual(held.want);
        });
    }
});

describe("writeQuery", () => {
    it("writes nothing for the default query", () => {
        expect(writeQuery(defaultQuery).toString()).toBe("");
        expect(searchOf(defaultQuery)).toBe("");
    });

    it("survives a round trip", () => {
        const query: TemplatesQuery = {
            scope: "global",
            pageKind: "hub",
            search: "mug",
            sort: { field: "createdAt", desc: true },
        };
        expect(readQuery(writeQuery(query))).toEqual(query);
        expect(searchOf(query).startsWith("?")).toBe(true);
    });

    it("writes every facet under its own key, in the order the address has always had", () => {
        const query: TemplatesQuery = {
            scope: "global",
            pageKind: "hub",
            search: "mug kiln",
            sort: { field: "createdAt", desc: true },
        };
        expect(searchOf(query)).toBe("?scope=global&kind=hub&q=mug+kiln&sort=createdAt%3Adesc");
    });

    it("never carries the action the palette sends", () => {
        expect(writeQuery(readQuery(new URLSearchParams("action=new&kind=guide"))).toString()).toBe("kind=guide");
    });
});

describe("narrowed", () => {
    it("knows when a filter hides rows", () => {
        expect(narrowed(defaultQuery)).toBe(false);
        expect(narrowed({ ...defaultQuery, search: "kiln" })).toBe(true);
        expect(narrowed({ ...defaultQuery, pageKind: "guide" })).toBe(true);
        expect(narrowed({ ...defaultQuery, scope: "global" })).toBe(true);
        expect(narrowed({ ...defaultQuery, sort: sorts.newest })).toBe(false);
    });
});

describe("sortChoice", () => {
    it("names the sort the toolbar shows", () => {
        expect(sortChoice(null)).toBe("nameAsc");
        expect(sortChoice(sorts.nameDesc)).toBe("nameDesc");
        expect(sortChoice(sorts.newest)).toBe("newest");
        expect(sortChoice(sorts.oldest)).toBe("oldest");
    });
});
