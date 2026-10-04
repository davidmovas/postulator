import { describe, expect, it } from "vitest";

import type { RunsQuery } from "./params.js";
import {
    defaultQuery,
    filterOf,
    itemSearchOf,
    narrowed,
    readItemStatus,
    readQuery,
    searchOf,
    writeQuery,
} from "./params.js";

describe("readQuery", () => {
    it("reads the filters the list endpoint supports", () => {
        const query = readQuery(new URLSearchParams("status=running&kind=generate&sort=status:desc"));
        expect(query).toStrictEqual({ status: "running", kind: "generate", sort: { field: "status", desc: true } });
    });

    it("drops a status outside the vocabulary", () => {
        expect(readQuery(new URLSearchParams("status=done")).status).toBe("");
        expect(readQuery(new URLSearchParams("status=needs_human")).status).toBe("");
        expect(readQuery(new URLSearchParams("status=skipped")).status).toBe("");
    });

    it("drops a kind outside the vocabulary", () => {
        expect(readQuery(new URLSearchParams("kind=rewrite")).kind).toBe("");
    });

    it("accepts only the sort fields the backend declares for runs", () => {
        expect(readQuery(new URLSearchParams("sort=createdAt:asc")).sort).toStrictEqual({
            field: "createdAt",
            desc: false,
        });
        expect(readQuery(new URLSearchParams("sort=path:asc")).sort).toBeNull();
        expect(readQuery(new URLSearchParams("sort=name:asc")).sort).toBeNull();
        expect(readQuery(new URLSearchParams("sort=status")).sort).toBeNull();
    });
});

describe("writeQuery", () => {
    it("round-trips through readQuery", () => {
        const query: RunsQuery = { status: "paused", kind: "audit", sort: { field: "createdAt", desc: true } };
        expect(readQuery(writeQuery(query))).toStrictEqual(query);
    });

    it("omits every default", () => {
        expect(writeQuery(defaultQuery).toString()).toBe("");
        expect(searchOf(defaultQuery)).toBe("");
    });

    it("writes every filter under its own key, in the order the address has always had", () => {
        const query: RunsQuery = { status: "paused", kind: "audit", sort: { field: "createdAt", desc: true } };
        expect(searchOf(query)).toBe("?status=paused&kind=audit&sort=createdAt%3Adesc");
    });
});

describe("filterOf", () => {
    it("carries only the narrowed fields", () => {
        expect(filterOf("s1", defaultQuery)).toStrictEqual({ siteId: "s1" });
        expect(filterOf("s1", { status: "failed", kind: "", sort: null })).toStrictEqual({
            siteId: "s1",
            status: "failed",
        });
    });

    it("never carries a cursor or a limit", () => {
        const filter = filterOf("s1", { status: "failed", kind: "generate", sort: null });
        expect(Object.keys(filter).sort()).toStrictEqual(["kind", "siteId", "status"]);
    });
});

describe("narrowed", () => {
    it("ignores the sort", () => {
        expect(narrowed({ status: "", kind: "", sort: { field: "status", desc: false } })).toBe(false);
        expect(narrowed({ status: "running", kind: "", sort: null })).toBe(true);
    });
});

describe("item status", () => {
    it("reads only a declared item status", () => {
        expect(readItemStatus(new URLSearchParams("item=waiting"))).toBe("waiting");
        expect(readItemStatus(new URLSearchParams("item=needs_human"))).toBe("");
        expect(readItemStatus(new URLSearchParams())).toBe("");
    });

    it("writes nothing when nothing is selected", () => {
        expect(itemSearchOf("")).toBe("");
        expect(itemSearchOf("failed")).toBe("?item=failed");
    });
});

describe("the start action", () => {
    it("is dropped by writeQuery, so a reload cannot reopen the drawer", () => {
        const query = readQuery(new URLSearchParams("action=new&kind=generate"));
        expect(writeQuery(query).has("action")).toBe(false);
        expect(writeQuery(query).get("kind")).toBe("generate");
    });
});
