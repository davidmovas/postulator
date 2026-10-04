import { describe, expect, it } from "vitest";

import type { PreviewReport } from "../../data/types.js";
import { columnsBySheet, sheetsIn } from "./preview.js";

function report(part: Partial<PreviewReport>): PreviewReport {
    return {
        columns: null,
        pages: null,
        entities: null,
        groups: null,
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
                pages: [{ path: "/a/", title: "A", keywords: null, wpType: "page", action: "create" }],
            },
            want: [],
        },
        {
            name: "a workbook, in the order its rows came",
            part: {
                columns: [{ sheet: "Catalog", header: "URL", use: "field", field: "path" }],
                pages: [
                    { sheet: "Catalog", path: "/a/", title: "A", keywords: null, wpType: "page", action: "create" },
                    { sheet: "Forms", path: "/a/b/", title: "B", keywords: null, wpType: "page", action: "create" },
                ],
                entities: [
                    { sheet: "Peptides", name: "BPC-157", kind: "product", keywords: null, anchors: null, action: "create" },
                ],
                groups: [{ sheet: "Forms", path: ["A"], rows: 1 }],
            },
            want: ["Catalog", "Forms", "Peptides"],
        },
    ])("lists $name", ({ part, want }) => {
        expect(sheetsIn(report(part))).toEqual(want);
    });
});

describe("the column uses of each sheet", () => {
    it("keeps each sheet's columns together in the order they came", () => {
        const grouped = columnsBySheet([
            { sheet: "Catalog", header: "URL", use: "field", field: "path" },
            { sheet: "Catalog", header: "Root Entity", use: "level" },
            { sheet: "Peptides", header: "Entity", use: "field", field: "entity" },
        ]);
        expect(grouped.map((held) => [held.sheet, held.columns.map((column) => column.header)])).toEqual([
            ["Catalog", ["URL", "Root Entity"]],
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
