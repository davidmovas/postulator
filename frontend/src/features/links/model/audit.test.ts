import { describe, expect, it } from "vitest";

import { potteryEdges, potteryEntities } from "../../graph/model/fixture.js";
import { buildGraphIndex } from "../../graph/model/index.js";
import { rows, severityOf, showCounts, tintByEntity } from "./audit.js";
import { auditRows } from "./fixture.js";
import { defaultQuery } from "./params.js";

const index = buildGraphIndex(potteryEntities, potteryEdges);

function paths(list: readonly { path: string }[]): string[] {
    return list.map((held) => held.path);
}

describe("rows", () => {
    it("sorts by severity, the worst page first, then by path", () => {
        expect(paths(rows(auditRows, defaultQuery, null))).toStrictEqual([
            "/pottery/mugs/",
            "/pottery/glazing/",
            "/pottery/",
            "/blog/diary/",
            "/pottery/mugs/350/",
            "/tea/",
        ]);
    });

    it("sorts by path when asked", () => {
        expect(paths(rows(auditRows, { ...defaultQuery, sort: "path" }, null))).toStrictEqual([
            "/blog/diary/",
            "/pottery/",
            "/pottery/glazing/",
            "/pottery/mugs/",
            "/pottery/mugs/350/",
            "/tea/",
        ]);
    });

    it("filters by what to show", () => {
        expect(paths(rows(auditRows, { ...defaultQuery, show: "missing" }, null))).toStrictEqual(["/pottery/mugs/", "/pottery/glazing/", "/pottery/"]);
        expect(paths(rows(auditRows, { ...defaultQuery, show: "missingRequired" }, null))).toStrictEqual(["/pottery/mugs/"]);
        expect(paths(rows(auditRows, { ...defaultQuery, show: "blocked" }, null))).toStrictEqual(["/pottery/mugs/"]);
        expect(paths(rows(auditRows, { ...defaultQuery, show: "offGraph" }, null))).toStrictEqual(["/pottery/mugs/"]);
        expect(paths(rows(auditRows, { ...defaultQuery, show: "orphans" }, null))).toStrictEqual(["/pottery/glazing/", "/blog/diary/"]);
        expect(paths(rows(auditRows, { ...defaultQuery, show: "skipped" }, null))).toStrictEqual(["/blog/diary/", "/tea/"]);
    });

    it("filters by status", () => {
        expect(paths(rows(auditRows, { ...defaultQuery, status: "exists" }, null))).toStrictEqual(["/blog/diary/"]);
    });

    it("keeps the pages of an entity's placement subtree", () => {
        expect(paths(rows(auditRows, { ...defaultQuery, entity: "mugs" }, index))).toStrictEqual(["/pottery/mugs/", "/pottery/mugs/350/"]);
        expect(paths(rows(auditRows, { ...defaultQuery, entity: "pottery" }, index))).toStrictEqual([
            "/pottery/mugs/",
            "/pottery/glazing/",
            "/pottery/",
            "/pottery/mugs/350/",
        ]);
        expect(paths(rows(auditRows, { ...defaultQuery, entity: "mugs" }, null))).toStrictEqual(["/pottery/mugs/"]);
    });
});

describe("showCounts", () => {
    it("counts every filter once", () => {
        expect(showCounts(auditRows)).toStrictEqual({ all: 6, missing: 3, missingRequired: 1, blocked: 1, offGraph: 1, orphans: 2, skipped: 2 });
    });
});

describe("severityOf and tintByEntity", () => {
    it("grades a page", () => {
        expect(severityOf(auditRows[1])).toBe("danger");
        expect(severityOf(auditRows[0])).toBe("warn");
        expect(severityOf(auditRows[2])).toBe("ok");
        expect(severityOf(auditRows[4])).toBe("muted");
    });

    it("tints every mapped entity by its worst page", () => {
        const tint = tintByEntity([...auditRows, { ...auditRows[2], pageId: "p-m350-b", path: "/mugs-b/", missingRequired: 1, missing: 1 }]);
        expect(tint.get("mugs")).toBe("danger");
        expect(tint.get("pottery")).toBe("warn");
        expect(tint.get("m350")).toBe("danger");
        expect(tint.get("tea")).toBe("muted");
        expect(tint.has("")).toBe(false);
    });
});
