import { describe, expect, it } from "vitest";

import {
    defaultQuery,
    filterOf,
    formatSort,
    narrowed,
    nextSort,
    parseSort,
    readQuery,
    searchOf,
    wantsNew,
    writeQuery,
} from "./params.js";

describe("parseSort", () => {
    it("accepts the two fields the backend declares", () => {
        expect(parseSort("path:asc")).toStrictEqual({ field: "path", desc: false });
        expect(parseSort("createdAt:desc")).toStrictEqual({ field: "createdAt", desc: true });
    });

    it("refuses a field the backend does not declare", () => {
        expect(parseSort("status:asc")).toBeNull();
        expect(parseSort("path:sideways")).toBeNull();
        expect(parseSort("path")).toBeNull();
        expect(parseSort("")).toBeNull();
        expect(parseSort(null)).toBeNull();
    });
});

describe("readQuery", () => {
    it("reads every filter the list endpoint supports", () => {
        const query = readQuery(
            new URLSearchParams("view=tree&status=published&entity=e1&unmapped=1&prefix=/shop/&sort=path:desc"),
        );
        expect(query).toStrictEqual({
            view: "tree",
            status: "published",
            entityId: "e1",
            unmapped: true,
            pathPrefix: "/shop/",
            sort: { field: "path", desc: true },
        });
    });

    it("drops a status that is not in the vocabulary", () => {
        expect(readQuery(new URLSearchParams("status=done")).status).toBe("");
    });

    it("falls back to the table view", () => {
        expect(readQuery(new URLSearchParams("view=canvas")).view).toBe("table");
        expect(readQuery(new URLSearchParams())).toStrictEqual(defaultQuery);
    });
});

describe("writeQuery", () => {
    it("round-trips through readQuery", () => {
        const query = {
            view: "tree" as const,
            status: "archived",
            entityId: "e9",
            unmapped: true,
            pathPrefix: "/a/",
            sort: { field: "createdAt" as const, desc: false },
        };
        expect(readQuery(writeQuery(query))).toStrictEqual(query);
    });

    it("writes nothing for the default query", () => {
        expect(writeQuery(defaultQuery).toString()).toBe("");
        expect(searchOf(defaultQuery)).toBe("");
    });
});

describe("filterOf", () => {
    it("omits every filter that is not set, so the query key stays stable", () => {
        expect(filterOf("site", defaultQuery)).toStrictEqual({ siteId: "site" });
    });

    it("carries the filters that are set", () => {
        expect(
            filterOf("site", { ...defaultQuery, status: "planned", entityId: "e1", unmapped: true, pathPrefix: "/x/" }),
        ).toStrictEqual({ siteId: "site", status: "planned", entityId: "e1", unmapped: true, pathPrefix: "/x/" });
    });
});

describe("narrowed", () => {
    it("is false only for the untouched query", () => {
        expect(narrowed(defaultQuery)).toBe(false);
        expect(narrowed({ ...defaultQuery, unmapped: true })).toBe(true);
        expect(narrowed({ ...defaultQuery, view: "tree" })).toBe(false);
    });
});

describe("nextSort", () => {
    it("cycles ascending, descending, then back to the backend default", () => {
        expect(nextSort(null, "path")).toStrictEqual({ field: "path", desc: false });
        expect(nextSort({ field: "path", desc: false }, "path")).toStrictEqual({ field: "path", desc: true });
        expect(nextSort({ field: "path", desc: true }, "path")).toBeNull();
    });

    it("restarts ascending when the field changes", () => {
        expect(nextSort({ field: "path", desc: true }, "createdAt")).toStrictEqual({
            field: "createdAt",
            desc: false,
        });
    });
});

describe("formatSort", () => {
    it("renders the segment the query key uses", () => {
        expect(formatSort({ field: "path", desc: true })).toBe("path:desc");
        expect(formatSort(null)).toBe("");
    });
});

describe("wantsNew", () => {
    it("reads the create action the palette sends", () => {
        expect(wantsNew(new URLSearchParams("action=new"))).toBe(true);
        expect(wantsNew(new URLSearchParams("action=edit"))).toBe(false);
        expect(wantsNew(new URLSearchParams())).toBe(false);
    });

    it("is dropped by writeQuery, so a reload cannot reopen the form", () => {
        const query = readQuery(new URLSearchParams("action=new&status=planned"));
        expect(writeQuery(query).has("action")).toBe(false);
        expect(writeQuery(query).get("status")).toBe("planned");
    });
});
