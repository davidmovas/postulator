import { describe, expect, it } from "vitest";

import type { RunsQuery } from "./params.js";
import {
    defaultQuery,
    filterOf,
    formatSort,
    itemSearchOf,
    narrowed,
    nextSort,
    parseSort,
    readItemStatus,
    readQuery,
    searchOf,
    wantsNew,
    writeQuery,
} from "./params.js";

describe("parseSort", () => {
    it("accepts only the two fields the backend declares for runs", () => {
        expect(parseSort("createdAt:asc")).toStrictEqual({ field: "createdAt", desc: false });
        expect(parseSort("status:desc")).toStrictEqual({ field: "status", desc: true });
    });

    it("refuses a field runs do not declare", () => {
        expect(parseSort("path:asc")).toBeNull();
        expect(parseSort("name:asc")).toBeNull();
        expect(parseSort("status:sideways")).toBeNull();
        expect(parseSort("status")).toBeNull();
        expect(parseSort(null)).toBeNull();
    });
});

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

describe("nextSort", () => {
    it("cycles ascending, descending, none", () => {
        const first = nextSort(null, "status");
        expect(first).toStrictEqual({ field: "status", desc: false });
        const second = nextSort(first, "status");
        expect(second).toStrictEqual({ field: "status", desc: true });
        expect(nextSort(second, "status")).toBeNull();
    });

    it("restarts on another field", () => {
        expect(nextSort({ field: "status", desc: true }, "createdAt")).toStrictEqual({
            field: "createdAt",
            desc: false,
        });
    });
});

describe("formatSort", () => {
    it("answers an empty string for no sort", () => {
        expect(formatSort(null)).toBe("");
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

describe("wantsNew", () => {
    it("reads the start action the palette sends", () => {
        expect(wantsNew(new URLSearchParams("action=new"))).toBe(true);
        expect(wantsNew(new URLSearchParams("action=start"))).toBe(false);
        expect(wantsNew(new URLSearchParams())).toBe(false);
    });

    it("is dropped by writeQuery, so a reload cannot reopen the drawer", () => {
        const query = readQuery(new URLSearchParams("action=new&kind=generate"));
        expect(writeQuery(query).has("action")).toBe(false);
        expect(writeQuery(query).get("kind")).toBe("generate");
    });
});
