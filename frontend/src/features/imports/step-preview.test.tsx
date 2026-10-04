import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { PreviewReport } from "../../data/types.js";
import { StepPreview } from "./step-preview.js";

type Entity = NonNullable<PreviewReport["entities"]>[number];
type Page = NonNullable<PreviewReport["pages"]>[number];

function entity(part: Pick<Entity, "name" | "kind"> & Partial<Entity>): Entity {
    return { keywords: null, anchors: null, action: "create", ...part };
}

function page(part: Pick<Page, "path" | "title"> & Partial<Page>): Page {
    return { keywords: null, wpType: "page", categories: [], action: "create", ...part };
}

const said = copy.imports.preview;

const workbook: PreviewReport = {
    columns: [
        { sheet: "Catalog", header: "URL", use: "field", field: "path" },
        { sheet: "Catalog", header: "Root Entity", use: "level" },
        { sheet: "Catalog", header: "Category", use: "level" },
        { sheet: "Catalog", header: "Subcategory", use: "level" },
        { sheet: "Compounds", header: "Entity", use: "field", field: "entity" },
    ],
    pages: [
        page({ sheet: "Catalog", path: "/peptides/", title: "Peptides" }),
        page({
            sheet: "Catalog",
            path: "/peptides/bpc-157-liquid/",
            title: "BPC-157 Liquid",
            categories: ["Healing", "Liquid"],
        }),
        page({
            sheet: "Compounds",
            path: "/peptides/bpc-157/",
            title: "BPC-157",
            entity: "BPC-157",
            categories: ["Healing"],
            action: "update",
        }),
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
    categories: [
        { sheet: "Catalog", path: ["Healing"], action: "match", rows: 2 },
        { sheet: "Catalog", path: ["Healing", "Liquid"], action: "create", rows: 1 },
        { sheet: "Compounds", path: ["Healing"], action: "match", rows: 1 },
        { path: ["Old"], action: "delete", rows: 0 },
    ],
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
        pages: (workbook.pages ?? []).filter((held) => held.sheet === "Catalog"),
        entities: (workbook.entities ?? []).filter((held) => held.sheet === "Catalog"),
        categories: (workbook.categories ?? []).filter((held) => held.sheet === "Catalog"),
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

function categoryRows(): HTMLElement[] {
    return [...document.querySelectorAll<HTMLElement>("[data-preview-category]")];
}

function trailOf(row: HTMLElement): { names: string[]; states: (string | undefined)[] } {
    const trail = within(row).getByRole("list", { name: copy.categories.trail });
    const chips = [...trail.querySelectorAll("li")];
    return {
        names: chips.map((chip) => chip.textContent?.replace("›", "") ?? ""),
        states: chips.map((chip) => chip.dataset["categoryState"]),
    };
}

describe("the preview of a workbook", () => {
    it("names the sheet of every row when more than one sheet is read", () => {
        show(workbook);
        expect(screen.getByText(said.columnSheet)).toBeDefined();
        expect(within(rowFor("/peptides/bpc-157/")).getByText("Compounds")).toBeDefined();
        expect(within(rowFor("/peptides/")).getByText("Catalog")).toBeDefined();
    });

    it("leaves the sheet out of a preview that read one sheet", () => {
        show(oneSheet());
        expect(screen.queryByText(said.columnSheet)).toBeNull();
    });

    it("says what each column of each sheet became, a root level apart from a category level", () => {
        show(workbook);
        const uses = screen.getByRole("region", { name: said.columns });
        expect(within(uses).getByText("Catalog")).toBeDefined();
        expect(within(uses).getByText("Compounds")).toBeDefined();
        expect(within(uses).getByText("Root Entity").parentElement?.textContent).toContain(copy.imports.rootLevel);
        expect(within(uses).getByText("Category").parentElement?.textContent).toContain(copy.imports.columnUses.level);
        expect(within(uses).getByText("Subcategory").parentElement?.textContent).toContain(copy.imports.columnUses.level);
    });

    it("lists the groups with their trail, the page each found, their rows and their sheet", () => {
        show(workbook);
        open(said.groups);
        const found = within(rowFor("Peptides › BPC-157"));
        expect(found.getByText("/peptides/bpc-157/")).toBeDefined();
        expect(found.getByText("3")).toBeDefined();
        expect(found.getByText("Catalog")).toBeDefined();
        expect(within(rowFor("Research")).getByText(said.noPage)).toBeDefined();
    });

    it("counts the groups and the categories on their segments", () => {
        show(workbook);
        expect(screen.getByRole("radio", { name: `${said.groups} 2` })).toBeDefined();
        expect(screen.getByRole("radio", { name: `${said.categories} 4` })).toBeDefined();
    });

    it("goes on to the apply step", () => {
        const onNext = show(workbook);
        fireEvent.click(screen.getByRole("button", { name: copy.imports.next }));
        expect(onNext).toHaveBeenCalledOnce();
    });

    it("keeps the next step shut while the preview reports errors", () => {
        show({ ...workbook, errors: [{ sheet: "Catalog", row: 4, code: "scope_clash", message: "" }] });
        expect(screen.getByRole("button", { name: copy.imports.next }).hasAttribute("disabled")).toBe(true);
        expect(screen.getByText(said.blocked)).toBeDefined();
    });
});

describe("the categories of a preview", () => {
    it("lists every category with its trail, the rows it files, what happens to it and its sheet", () => {
        show(workbook);
        open(said.categories);
        expect(screen.getByText(said.categoriesHint)).toBeDefined();

        const [healing, liquid, again, old] = categoryRows();
        if (healing === undefined || liquid === undefined || again === undefined || old === undefined) {
            throw new Error("the preview lists fewer categories than it was given");
        }
        expect(trailOf(liquid)).toStrictEqual({ names: ["Healing", "Liquid"], states: ["known", "becomes"] });
        expect(within(liquid).getByText(copy.imports.categoryActions.create)).toBeDefined();
        expect(within(liquid).getByText("1")).toBeDefined();
        expect(within(liquid).getByText("Catalog")).toBeDefined();

        expect(trailOf(healing).states).toStrictEqual(["known"]);
        expect(within(healing).getByText(copy.imports.categoryActions.match)).toBeDefined();
        expect(within(again).getByText("Compounds")).toBeDefined();

        expect(trailOf(old).names).toStrictEqual(["Old"]);
        expect(within(old).getByText(copy.imports.categoryActions.delete)).toBeDefined();
        expect(old.dataset["previewCategory"]).toBe("delete");
    });

    it("tallies what the import does to the categories in words", () => {
        show(workbook);
        open(said.categories);
        const tallied = [...document.querySelectorAll<HTMLElement>("[data-tally]")].map((chip) => [
            chip.dataset["tally"],
            chip.textContent,
        ]);
        expect(tallied).toStrictEqual([
            ["create", `1${copy.imports.categoryActions.create}`],
            ["match", `2${copy.imports.categoryActions.match}`],
            ["delete", `1${copy.imports.categoryActions.delete}`],
        ]);
    });

    it("says how to file pages when the sheet files none", () => {
        show({ ...workbook, categories: [] });
        open(said.categories);
        expect(screen.getByText(said.noCategories)).toBeDefined();
        expect(categoryRows()).toHaveLength(0);
    });

    it("shows each page's chain, the levels the import creates drawn apart", () => {
        show(workbook);
        expect(screen.getByText(said.columnCategories)).toBeDefined();
        expect(trailOf(rowFor("/peptides/bpc-157-liquid/"))).toStrictEqual({
            names: ["Healing", "Liquid"],
            states: ["known", "becomes"],
        });
        expect(trailOf(rowFor("/peptides/bpc-157/")).states).toStrictEqual(["known"]);
        expect(within(rowFor("/peptides/")).queryByRole("list", { name: copy.categories.trail })).toBeNull();
    });

    it("marks no entity as a category", () => {
        show(workbook);
        open(said.entities);
        expect(screen.queryByRole("list", { name: copy.categories.trail })).toBeNull();
        expect(within(rowFor("Peptides")).getByText(copy.graph.kinds.hub)).toBeDefined();
    });
});
