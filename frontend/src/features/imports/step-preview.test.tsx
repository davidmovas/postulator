import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { PreviewReport } from "../../data/types.js";
import { StepPreview } from "./step-preview.js";

type Entity = NonNullable<PreviewReport["entities"]>[number];

function entity(part: Pick<Entity, "name" | "kind"> & Partial<Entity>): Entity {
    return { keywords: null, anchors: null, action: "create", ...part };
}

const workbook: PreviewReport = {
    columns: [
        { sheet: "Catalog", header: "URL", use: "field", field: "path" },
        { sheet: "Catalog", header: "Root Entity", use: "level" },
        { sheet: "Catalog", header: "Category", use: "ignored" },
        { sheet: "Compounds", header: "Entity", use: "field", field: "entity" },
    ],
    pages: [
        {
            sheet: "Catalog",
            path: "/peptides/",
            title: "Peptides",
            keywords: null,
            wpType: "page",
            categories: [],
            action: "create",
        },
        {
            sheet: "Compounds",
            path: "/peptides/bpc-157/",
            title: "BPC-157",
            keywords: null,
            wpType: "page",
            entity: "BPC-157",
            categories: [],
            action: "update",
        },
    ],
    entities: [
        entity({ sheet: "Catalog", name: "Peptides", kind: "hub" }),
        entity({ sheet: "Catalog", name: "Research", kind: "hub" }),
        entity({ sheet: "Compounds", name: "BPC-157", parent: "Peptides", kind: "product" }),
    ],
    groups: [
        { sheet: "Catalog", path: ["Peptides", "BPC-157"], page: "/peptides/bpc-157/", rows: 3 },
        { sheet: "Catalog", path: ["Research"], rows: 1 },
    ],
    categories: [],
    edges: [{ sheet: "Compounds", from: "BPC-157", to: "Peptides", kind: "parent", action: "create" }],
    warnings: null,
    errors: null,
    cannibalization: null,
    skipped: 0,
};

function oneSheet(): PreviewReport {
    return {
        ...workbook,
        columns: (workbook.columns ?? []).filter((column) => column.sheet === "Catalog"),
        pages: (workbook.pages ?? []).filter((page) => page.sheet === "Catalog"),
        entities: (workbook.entities ?? []).filter((entity) => entity.sheet === "Catalog"),
        edges: [],
    };
}

function show(report: PreviewReport, onNext = vi.fn()) {
    render(<StepPreview report={report} onBack={() => {}} onNext={onNext} />);
    return onNext;
}

function open(segment: string): void {
    fireEvent.click(screen.getByRole("radio", { name: new RegExp(`^${segment} `) }));
}

function rowFor(text: string): HTMLElement {
    const held = screen.getByText(text).closest('[role="row"]');
    if (held === null) {
        throw new Error("no row carries " + text);
    }
    return held as HTMLElement;
}

describe("the preview of a workbook", () => {
    it("names the sheet of every row when more than one sheet is read", () => {
        show(workbook);
        expect(screen.getByText(copy.imports.preview.columnSheet)).toBeDefined();
        expect(within(rowFor("/peptides/bpc-157/")).getByText("Compounds")).toBeDefined();
        expect(within(rowFor("/peptides/")).getByText("Catalog")).toBeDefined();
    });

    it("leaves the sheet out of a preview that read one sheet", () => {
        show(oneSheet());
        expect(screen.queryByText(copy.imports.preview.columnSheet)).toBeNull();
    });

    it("says what each column of each sheet became, a Category column ignored", () => {
        show(workbook);
        const uses = screen.getByRole("region", { name: copy.imports.preview.columns });
        expect(within(uses).getByText("Catalog")).toBeDefined();
        expect(within(uses).getByText("Compounds")).toBeDefined();
        expect(within(uses).getByText("Root Entity").parentElement?.textContent).toContain(copy.imports.columnUses.level);
        expect(within(uses).getByText("Category").parentElement?.textContent).toContain(copy.imports.columnUses.ignored);
    });

    it("lists the groups with their trail, the page each found, their rows and their sheet", () => {
        show(workbook);
        open(copy.imports.preview.groups);
        const found = within(rowFor("Peptides › BPC-157"));
        expect(found.getByText("/peptides/bpc-157/")).toBeDefined();
        expect(found.getByText("3")).toBeDefined();
        expect(found.getByText("Catalog")).toBeDefined();
        expect(within(rowFor("Research")).getByText(copy.imports.preview.noPage)).toBeDefined();
    });

    it("counts the groups on their segment", () => {
        show(workbook);
        expect(screen.getByRole("radio", { name: `${copy.imports.preview.groups} 2` })).toBeDefined();
    });

    it("goes on to the apply step", () => {
        const onNext = show(workbook);
        fireEvent.click(screen.getByRole("button", { name: copy.imports.next }));
        expect(onNext).toHaveBeenCalledOnce();
    });

    it("keeps the next step shut while the preview reports errors", () => {
        show({ ...workbook, errors: [{ sheet: "Catalog", row: 4, code: "scope_clash", message: "" }] });
        expect(screen.getByRole("button", { name: copy.imports.next }).hasAttribute("disabled")).toBe(true);
        expect(screen.getByText(copy.imports.preview.blocked)).toBeDefined();
    });
});
