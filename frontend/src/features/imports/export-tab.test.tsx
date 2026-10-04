import { screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { ImportFinding } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";

interface Written {
    path: string;
    format: string;
    pages: number;
    entities: number;
    warnings: ImportFinding[] | null;
}

const held: { written: Written | undefined } = { written: undefined };

vi.mock("../../data/hooks/imports.js", () => ({
    useExportSite: () => ({
        data: held.written,
        error: null,
        isPending: false,
        mutate: () => {},
        reset: () => {},
    }),
}));

vi.mock("../../data/hooks/sites.js", () => ({
    useSite: () => ({ data: undefined, isPending: false, error: null }),
}));

vi.mock("../../data/host.js", () => ({
    pickSaveFile: () => Promise.resolve(null),
}));

const { ExportTab } = await import("./export-tab.js");

function exported(warnings: ImportFinding[] | null): Written {
    return { path: "C:/out/crema-bench-export.xlsx", format: "xlsx", pages: 40, entities: 12, warnings };
}

beforeEach(() => {
    held.written = undefined;
});

describe("the export", () => {
    it("says in words what the written sheet could not carry", () => {
        held.written = exported([
            {
                row: 7,
                code: "category_chain_cut",
                message: "the page /a/b/c/d/ is filed under A › B › C › D; the file keeps its first 3 levels",
            },
        ]);
        renderScreen(<ExportTab siteId="s1" />);

        const notes = document.querySelector<HTMLElement>("[data-export-warnings]");
        if (notes === null) {
            throw new Error("the export shows no warnings");
        }
        expect(within(notes).getByText(copy.imports.export.warnings)).toBeDefined();
        expect(within(notes).getByText(copy.imports.findings.category_chain_cut)).toBeDefined();
        expect(within(notes).getByText(copy.imports.preview.row(7))).toBeDefined();
        expect(within(notes).getByText(/keeps its first 3 levels$/)).toBeDefined();
        expect(within(notes).queryByText("category_chain_cut")).toBeNull();
    });

    it.each([[[]], [null]])("adds nothing to a clean export (%j)", (warnings) => {
        held.written = exported(warnings);
        renderScreen(<ExportTab siteId="s1" />);
        expect(screen.getByText(copy.imports.export.done(40, 12))).toBeDefined();
        expect(document.querySelector("[data-export-warnings]")).toBeNull();
    });
});
