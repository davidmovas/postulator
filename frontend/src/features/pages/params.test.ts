import { describe, expect, it } from "vitest";

import { defaultQuery, filterOf, narrowed, readQuery, readTab, searchOf, withTab, writeQuery } from "./params.js";

describe("readQuery", () => {
    it("reads every filter the list endpoint supports", () => {
        const query = readQuery(
            new URLSearchParams(
                "view=tree&status=published&entity=e1&under=1&unmapped=1&prefix=/shop/&category=c7&sort=path:desc",
            ),
        );
        expect(query).toStrictEqual({
            view: "tree",
            status: "published",
            entityId: "e1",
            descendants: true,
            unmapped: true,
            pathPrefix: "/shop/",
            categoryId: "c7",
            sort: { field: "path", desc: true },
        });
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
            descendants: true,
            unmapped: true,
            pathPrefix: "/a/",
            categoryId: "c2",
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
            view: "tree" as const,
            status: "published",
            entityId: "e1",
            descendants: true,
            unmapped: true,
            pathPrefix: "/shop/",
            categoryId: "",
            sort: { field: "path" as const, desc: true },
        };
        expect(searchOf(query)).toBe(
            "?view=tree&status=published&entity=e1&under=1&unmapped=1&prefix=%2Fshop%2F&sort=path%3Adesc",
        );
        expect(searchOf({ ...query, categoryId: "c7" })).toBe(
            "?view=tree&status=published&entity=e1&under=1&unmapped=1&prefix=%2Fshop%2F&category=c7&sort=path%3Adesc",
        );
    });
});

describe("filterOf", () => {
    it("omits every filter that is not set, so the query key stays stable", () => {
        expect(filterOf("site", defaultQuery)).toStrictEqual({ siteId: "site" });
    });

    it("carries the filters that are set", () => {
        expect(
            filterOf("site", {
                ...defaultQuery,
                status: "planned",
                entityId: "e1",
                unmapped: true,
                pathPrefix: "/x/",
                categoryId: "c7",
            }),
        ).toStrictEqual({
            siteId: "site",
            status: "planned",
            entityId: "e1",
            unmapped: true,
            pathPrefix: "/x/",
            categoryId: "c7",
        });
    });

    it("asks for the pages under an entity only when an entity is chosen", () => {
        expect(filterOf("site", { ...defaultQuery, entityId: "e1", descendants: true })).toStrictEqual({
            siteId: "site",
            entityId: "e1",
            includeDescendants: true,
        });
        expect(filterOf("site", { ...defaultQuery, descendants: true })).toStrictEqual({ siteId: "site" });
    });

    it("asks for the pages of a category branch beside the other filters", () => {
        expect(filterOf("site", { ...defaultQuery, status: "published", categoryId: "c7" })).toStrictEqual({
            siteId: "site",
            status: "published",
            categoryId: "c7",
        });
    });
});

describe("narrowed", () => {
    it("is false only for the untouched query", () => {
        expect(narrowed(defaultQuery)).toBe(false);
        expect(narrowed({ ...defaultQuery, unmapped: true })).toBe(true);
        expect(narrowed({ ...defaultQuery, categoryId: "c7" })).toBe(true);
        expect(narrowed({ ...defaultQuery, view: "tree" })).toBe(false);
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
    });

    it("falls back to the details tab for anything it does not know", () => {
        expect(readTab(new URLSearchParams(""))).toBe("details");
        expect(readTab(new URLSearchParams("tab=telepathy"))).toBe("details");
    });

    it("writes every tab but the default one, so a plain link stays plain", () => {
        expect(withTab(new URLSearchParams("view=tree"), "preview").toString()).toBe("view=tree&tab=preview");
        expect(withTab(new URLSearchParams("view=tree"), "details").toString()).toBe("view=tree");
    });
});
