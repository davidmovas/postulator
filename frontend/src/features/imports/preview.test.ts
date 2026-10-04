import { describe, expect, it } from "vitest";

import type { PreviewReport } from "../../data/types.js";
import { categoryPathKey } from "../../domain/categories.js";
import { columnsBySheet, createdPaths, sheetsIn } from "./preview.js";

function report(part: Partial<PreviewReport>): PreviewReport {
    return {
        columns: null,
        pages: null,
        entities: null,
        groups: null,
        categories: null,
        edges: null,
        warnings: null,
        errors: null,
        cannibalization: null,
        skipped: 0,
        ...part,
    };
}

describe("the sheets a preview read", () => {
    it.each([
        { name: "nothing read", part: {}, want: [] },
        {
            name: "a csv, whose rows name no sheet",
            part: {
                pages: [{ path: "/a/", title: "A", keywords: null, wpType: "page", categories: null, action: "create" }],
            },
            want: [],
        },
        {
            name: "a workbook, in the order its rows came",
            part: {
                columns: [{ sheet: "Catalog", header: "URL", use: "field", field: "path" }],
                pages: [
                    { sheet: "Catalog", path: "/a/", title: "A", keywords: null, wpType: "page", categories: [], action: "create" },
                    { sheet: "Forms", path: "/a/b/", title: "B", keywords: null, wpType: "page", categories: [], action: "create" },
                ],
                entities: [
                    { sheet: "Peptides", name: "BPC-157", kind: "product", keywords: null, anchors: null, action: "create" },
                ],
                groups: [{ sheet: "Forms", path: ["A"], rows: 1 }],
            },
            want: ["Catalog", "Forms", "Peptides"],
        },
        {
            name: "a sheet that only files categories, and no sheet for the categories an apply removes",
            part: {
                categories: [
                    { sheet: "Catalog", path: ["Peptides"], action: "create", rows: 2 },
                    { path: ["Old"], action: "delete", rows: 0 },
                ],
            },
            want: ["Catalog"],
        },
    ])("lists $name", ({ part, want }) => {
        expect(sheetsIn(report(part))).toEqual(want);
    });
});

describe("the categories an import creates", () => {
    it("keys each created category by its whole path and leaves the found and removed ones out", () => {
        const created = createdPaths(
            report({
                categories: [
                    { sheet: "Catalog", path: ["Peptides"], action: "match", rows: 5 },
                    { sheet: "Catalog", path: ["Peptides", "Healing"], action: "create", rows: 3 },
                    { sheet: "Forms", path: ["Peptides", "Healing"], action: "match", rows: 1 },
                    { path: ["Old"], action: "delete", rows: 0 },
                ],
            }),
        );
        expect([...created]).toStrictEqual([categoryPathKey(["Peptides", "Healing"])]);
    });

    it("creates nothing from a preview that files no page", () => {
        expect(createdPaths(report({})).size).toBe(0);
    });
});

describe("the column uses of each sheet", () => {
    it("keeps each sheet's columns together in the order they came", () => {
        const grouped = columnsBySheet([
            { sheet: "Catalog", header: "URL", use: "field", field: "path" },
            { sheet: "Catalog", header: "Category", use: "level" },
            { sheet: "Peptides", header: "Entity", use: "field", field: "entity" },
        ]);
        expect(grouped.map((held) => [held.sheet, held.columns.map((column) => column.header)])).toEqual([
            ["Catalog", ["URL", "Category"]],
            ["Peptides", ["Entity"]],
        ]);
    });

    it("reads columns that name no sheet as one group", () => {
        expect(columnsBySheet([{ header: "URL", use: "field", field: "path" }])).toEqual([
            { sheet: "", columns: [{ header: "URL", use: "field", field: "path" }] },
        ]);
        expect(columnsBySheet([])).toEqual([]);
    });
});
