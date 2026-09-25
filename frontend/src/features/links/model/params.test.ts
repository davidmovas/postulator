import { describe, expect, it } from "vitest";

import { defaultQuery, narrowed, readQuery, searchOf, writeQuery } from "./params.js";

describe("readQuery and writeQuery", () => {
    it("reads the default from nothing", () => {
        expect(readQuery(new URLSearchParams())).toStrictEqual(defaultQuery);
        expect(defaultQuery).toStrictEqual({ show: "all", entity: "", status: "", sort: "severity" });
    });

    it("round-trips a full query", () => {
        const query = { show: "missingRequired" as const, entity: "mugs", status: "published", sort: "path" as const };
        const written = writeQuery(query).toString();
        expect(written).toBe("show=missingRequired&entity=mugs&status=published&sort=path");
        expect(readQuery(new URLSearchParams(written))).toStrictEqual(query);
    });

    it("reads the filters for what waits and for links to unpublished pages", () => {
        expect(readQuery(new URLSearchParams("show=pending")).show).toBe("pending");
        expect(readQuery(new URLSearchParams("show=unpublished")).show).toBe("unpublished");
    });

    it("falls back on values it does not know", () => {
        expect(readQuery(new URLSearchParams("show=bogus&status=nope&sort=weird"))).toStrictEqual(defaultQuery);
    });

    it("knows when the query narrows the rows", () => {
        expect(narrowed(defaultQuery)).toBe(false);
        expect(narrowed({ ...defaultQuery, show: "blocked" })).toBe(true);
        expect(narrowed({ ...defaultQuery, sort: "path" })).toBe(false);
        expect(searchOf({ ...defaultQuery, entity: "mugs" })).toBe("?entity=mugs");
    });
});
