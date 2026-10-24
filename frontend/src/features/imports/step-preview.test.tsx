import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { PreviewPage, PreviewReport } from "../../data/types.js";
import type { Choices } from "./choices.js";
import { noChoices, withChoice } from "./choices.js";
import { StepPreview } from "./step-preview.js";

beforeAll(() => {
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
    Element.prototype.scrollIntoView = () => undefined;
});

const candidates = [
    { pageId: "p-lab", path: "/lab/guides/", title: "Lab guides" },
    { pageId: "p-shop", path: "/shop/guides/", title: "Shop guides" },
];

function guides(part: Partial<PreviewPage>): PreviewReport {
    const page: PreviewPage = { sheet: "Catalog", path: "/guides/", title: "Guides", keywords: null, wpType: "page", action: "ambiguous", ...part };
    return { ...workbook, pages: [...(workbook.pages ?? []), page] };
}

type Entity = NonNullable<PreviewReport["entities"]>[number];

function entity(part: Pick<Entity, "name"> & Partial<Entity>): Entity {
    return { keywords: null, anchors: null, action: "create", ...part };
}

const workbook: PreviewReport = {
    columns: [
        { sheet: "Catalog", header: "URL", use: "field", field: "path" },
        { sheet: "Catalog", header: "Root Entity", use: "level" },
        { sheet: "Catalog", header: "Category", use: "trail" },
        { sheet: "Compounds", header: "Entity", use: "field", field: "entity" },
    ],
    pages: [
        {
            sheet: "Catalog",
            path: "/peptides/",
            title: "Peptides",
            keywords: null,
            wpType: "page",
            pageType: "Category",
            action: "create",
        },
        {
            sheet: "Compounds",
            path: "/peptides/bpc-157/",
            title: "BPC-157",
            keywords: null,
            wpType: "page",
            entity: "BPC-157",
            action: "update",
        },
    ],
    entities: [
        entity({ sheet: "Catalog", name: "Peptides" }),
        entity({ sheet: "Catalog", name: "Research" }),
        entity({ sheet: "Compounds", name: "BPC-157", parent: "Peptides" }),
    ],
    groups: [
        { sheet: "Catalog", path: ["Peptides", "BPC-157"], page: "/peptides/bpc-157/", rows: 3 },
        { sheet: "Catalog", path: ["Research"], rows: 1 },
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
        pages: (workbook.pages ?? []).filter((page) => page.sheet === "Catalog"),
        entities: (workbook.entities ?? []).filter((entity) => entity.sheet === "Catalog"),
        edges: [],
    };
}

function show(report: PreviewReport, onNext = vi.fn(), choices: Choices = noChoices, onPick = vi.fn()) {
    render(<StepPreview report={report} choices={choices} onPick={onPick} onBack={() => {}} onNext={onNext} />);
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

    it("shows the page type the sheet gives a page by the site's own name", () => {
        show(workbook);
        expect(screen.getByText(copy.imports.preview.columnType)).toBeDefined();
        expect(within(rowFor("/peptides/")).getByText("Category")).toBeDefined();
    });

    it("leaves the sheet out of a preview that read one sheet", () => {
        show(oneSheet());
        expect(screen.queryByText(copy.imports.preview.columnSheet)).toBeNull();
    });

    it("says what each column of each sheet became, a Category column the expected parent", () => {
        show(workbook);
        const uses = screen.getByRole("region", { name: copy.imports.preview.columns });
        expect(within(uses).getByText("Catalog")).toBeDefined();
        expect(within(uses).getByText("Compounds")).toBeDefined();
        expect(within(uses).getByText("Root Entity").parentElement?.textContent).toContain(copy.imports.columnUses.level);
        expect(within(uses).getByText("Category").parentElement?.textContent).toContain(copy.imports.columnUses.trail);
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

    it("names the rule each row found its page by, in words", () => {
        show(guides({ path: "/peptides/kpv/", action: "update", matchedBy: "slug" }));
        expect(screen.getByText(copy.imports.preview.columnMatched)).toBeDefined();
        const found = within(rowFor("/peptides/kpv/")).getByText(copy.imports.preview.matchRules.slug);
        expect(found.getAttribute("title")).toBe(copy.imports.preview.matchedTooltip(copy.imports.preview.matchedBy.slug));
    });

    it("offers the pages an ambiguous row matches and sends the one a person picks", () => {
        const onPick = vi.fn();
        show(guides({ rowKey: "Catalog#4", matchedBy: "title", candidates }), vi.fn(), noChoices, onPick);

        const pick = screen.getByRole("combobox", { name: copy.imports.preview.pickLabel("/guides/") });
        expect(pick.textContent).toContain(copy.imports.preview.pick);
        fireEvent.pointerDown(pick, { button: 0, ctrlKey: false, pointerType: "mouse" });
        fireEvent.click(screen.getByRole("option", { name: copy.imports.preview.candidate("/shop/guides/", "Shop guides") }));

        expect(onPick).toHaveBeenCalledWith("Catalog#4", "p-shop");
    });

    it("keeps the pick on a row the preview mapped, so it can change", () => {
        const picked = withChoice(noChoices, "Catalog#4", "p-lab");
        show(guides({ path: "/lab/guides/", rowKey: "Catalog#4", matchedBy: "mapped", action: "update", candidates }), vi.fn(), picked);

        const pick = screen.getByRole("combobox", { name: copy.imports.preview.pickLabel("/lab/guides/") });
        expect(pick.textContent).toContain(copy.imports.preview.candidate("/lab/guides/", "Lab guides"));
    });

    it("tells a person a row with nothing to pick it by needs a URL", () => {
        show(guides({ matchedBy: "title", candidates }));

        expect(within(rowFor("/guides/")).getByText(copy.imports.preview.needsUrl)).toBeDefined();
        expect(screen.queryByRole("combobox", { name: copy.imports.preview.pickLabel("/guides/") })).toBeNull();
    });

    it("keeps the next step shut while the preview reports errors", () => {
        show({ ...workbook, errors: [{ sheet: "Catalog", row: 4, code: "scope_clash", message: "" }] });
        expect(screen.getByRole("button", { name: copy.imports.next }).hasAttribute("disabled")).toBe(true);
        expect(screen.getByText(copy.imports.preview.blocked)).toBeDefined();
    });
});
