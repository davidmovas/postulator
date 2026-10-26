import { describe, expect, it } from "vitest";

import { defaultQuery, facetsOf, filterOf, narrowed, readQuery, readTab, searchOf, withTab, writeQuery } from "./params.js";

describe("readQuery", () => {
    it("reads every filter the list endpoint supports", () => {
        const query = readQuery(new URLSearchParams("status=published&entity=e1&under=1&unmapped=1&prefix=/shop/&sort=path:desc"));
        expect(query).toStrictEqual({
            status: "published",
            entityId: "e1",
            descendants: true,
            unmapped: true,
            pathPrefix: "/shop/",
            typeId: "",
            body: "",
            findings: false,
            branch: "",
            search: "",
            sort: { field: "path", desc: true },
        });
    });

    it("reads the type, the text, the findings, the branch and the search", () => {
        const query = readQuery(new URLSearchParams("type=t1&text=without&findings=1&branch=p1&q=mugs"));
        expect(query).toMatchObject({ typeId: "t1", body: "without", findings: true, branch: "p1", search: "mugs" });
        expect(readQuery(new URLSearchParams("text=maybe")).body).toBe("");
    });

    it("drops a status that is not in the vocabulary", () => {
        expect(readQuery(new URLSearchParams("status=done")).status).toBe("");
    });

    it("drops a sort on a field the backend does not declare for pages", () => {
        expect(readQuery(new URLSearchParams("sort=status:asc")).sort).toBeNull();
        expect(readQuery(new URLSearchParams("sort=createdAt:desc")).sort).toStrictEqual({
            field: "createdAt",
            desc: true,
        });
    });

    it("ignores the tree view an old address still asks for, since the tree is the Site map", () => {
        expect(readQuery(new URLSearchParams("view=tree"))).toStrictEqual(defaultQuery);
        expect(readQuery(new URLSearchParams())).toStrictEqual(defaultQuery);
    });
});

describe("writeQuery", () => {
    it("round-trips through readQuery", () => {
        const query = {
            status: "archived",
            entityId: "e9",
            descendants: true,
            unmapped: true,
            pathPrefix: "/a/",
            typeId: "t1",
            body: "with" as const,
            findings: true,
            branch: "p1",
            search: "mugs",
            sort: { field: "createdAt" as const, desc: false },
        };
        expect(readQuery(writeQuery(query))).toStrictEqual(query);
    });

    it("writes nothing for the default query", () => {
        expect(writeQuery(defaultQuery).toString()).toBe("");
        expect(searchOf(defaultQuery)).toBe("");
    });

    it("writes every filter under its own key, in the order the address has always had", () => {
        const query = {
            ...defaultQuery,
            status: "published",
            entityId: "e1",
            descendants: true,
            unmapped: true,
            pathPrefix: "/shop/",
            sort: { field: "path" as const, desc: true },
        };
        expect(searchOf(query)).toBe("?status=published&entity=e1&under=1&unmapped=1&prefix=%2Fshop%2F&sort=path%3Adesc");
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

    it("asks for the pages under an entity only when an entity is chosen", () => {
        expect(filterOf("site", { ...defaultQuery, entityId: "e1", descendants: true })).toStrictEqual({
            siteId: "site",
            entityId: "e1",
            includeDescendants: true,
        });
        expect(filterOf("site", { ...defaultQuery, descendants: true })).toStrictEqual({ siteId: "site" });
    });

    it("carries the type, the text, the findings, the branch and a trimmed search", () => {
        expect(
            filterOf("site", { ...defaultQuery, typeId: "t1", body: "without", findings: true, branch: "p1", search: "  mugs " }),
        ).toStrictEqual({ siteId: "site", pageTypeId: "t1", body: "without", findings: true, under: "p1", search: "mugs" });
        expect(filterOf("site", { ...defaultQuery, search: "   " })).toStrictEqual({ siteId: "site" });
    });

    it("hands the facets of a filter to a selection without its site", () => {
        expect(facetsOf({ siteId: "site", status: "planned", findings: true })).toStrictEqual({ status: "planned", findings: true });
    });
});

describe("narrowed", () => {
    it("is false only for the untouched query", () => {
        expect(narrowed(defaultQuery)).toBe(false);
        expect(narrowed({ ...defaultQuery, unmapped: true })).toBe(true);
        expect(narrowed({ ...defaultQuery, sort: { field: "path", desc: true } })).toBe(false);
    });
});

describe("the create action", () => {
    it("is dropped by writeQuery, so a reload cannot reopen the form", () => {
        const query = readQuery(new URLSearchParams("action=new&status=planned"));
        expect(writeQuery(query).has("action")).toBe(false);
        expect(writeQuery(query).get("status")).toBe("planned");
    });
});

describe("the drawer tab in the address", () => {
    it("reads a tab another screen asked for", () => {
        expect(readTab(new URLSearchParams("tab=preview"))).toBe("preview");
        expect(readTab(new URLSearchParams("tab=mapping"))).toBe("mapping");
        expect(readTab(new URLSearchParams("tab=model"))).toBe("model");
        expect(readTab(new URLSearchParams("tab=writer"))).toBe("writer");
    });

    it("falls back to the details tab for anything it does not know", () => {
        expect(readTab(new URLSearchParams(""))).toBe("details");
        expect(readTab(new URLSearchParams("tab=telepathy"))).toBe("details");
    });

    it("writes every tab but the default one, so a plain link stays plain", () => {
        expect(withTab(new URLSearchParams("status=planned"), "preview").toString()).toBe("status=planned&tab=preview");
        expect(withTab(new URLSearchParams("status=planned"), "details").toString()).toBe("status=planned");
    });
});
