import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { ImportMapping, ImportSheet } from "../../data/types.js";
import type { OptionsPanelProps } from "./options-panel.js";

const held: { mappings: ImportMapping[] } = { mappings: [] };

vi.mock("../../data/hooks/imports.js", () => ({
    useMappings: () => ({ data: { mappings: held.mappings }, isPending: false, error: null }),
}));

const { OptionsPanel } = await import("./options-panel.js");

function sheet(name: string, rows: number): ImportSheet {
    return { name, rows, headers: [], detected: { columns: {}, options: {}, createdAt: null, updatedAt: null } };
}

const workbook = [sheet("Catalog", 3), sheet("Compounds", 12), sheet("Readme", 0)];

function show(part: Partial<OptionsPanelProps> = {}) {
    const props: OptionsPanelProps = {
        siteId: "s1",
        sheets: workbook,
        chosen: ["Catalog", "Compounds"],
        active: "Catalog",
        settings: {
            columns: { path: "URL" },
            options: { levelColumns: ["Category"], noteColumns: [] },
            mappingId: "",
        },
        headers: ["URL", "Category", "Notes"],
        onChoose: vi.fn(),
        onOptions: vi.fn(),
        onHeaderless: vi.fn(),
        onSaved: vi.fn(),
        ...part,
    };
    render(<OptionsPanel {...props} />);
    return props;
}

beforeEach(() => {
    held.mappings = [];
});

describe("the options of a workbook import", () => {
    it("lists every sheet with its rows and says which ones are imported", () => {
        show();
        expect(screen.getByRole("switch", { name: "Catalog" })).toHaveProperty("checked", true);
        expect(screen.getByRole("switch", { name: "Readme" })).toHaveProperty("checked", false);
        expect(screen.getByText(copy.imports.columns.sheetRows(12))).toBeDefined();
        expect(screen.getByText(copy.imports.columns.reading(["Catalog", "Compounds"]))).toBeDefined();
        expect(screen.getByText("Importing Catalog and Compounds.")).toBeDefined();
    });

    it("turns a sheet on or off without touching the settings of any sheet", () => {
        const props = show();
        fireEvent.click(screen.getByRole("switch", { name: "Compounds" }));
        fireEvent.click(screen.getByRole("switch", { name: "Readme" }));
        expect(props.onChoose).toHaveBeenNthCalledWith(1, "Compounds", false);
        expect(props.onChoose).toHaveBeenNthCalledWith(2, "Readme", true);
        expect(props.onOptions).not.toHaveBeenCalled();
    });

    it("says whose settings the panels below hold", () => {
        show();
        expect(screen.getByText(copy.imports.columns.settingsOf("Catalog"))).toBeDefined();
    });

    it("shows no sheet list and no sheet name for a file with one sheet", () => {
        show({ sheets: [sheet("", 3)], chosen: [""], active: "" });
        expect(screen.queryByText(copy.imports.columns.sheets)).toBeNull();
        expect(screen.getByText(copy.imports.columns.levels)).toBeDefined();
    });

    it("hides the settings of a sheet once no sheet is on", () => {
        show({ chosen: [], active: "" });
        expect(screen.queryByText(copy.imports.columns.levels)).toBeNull();
    });

    it("explains that a Category column becomes a WordPress category and a Root Entity column a hub", () => {
        show();
        const hint = screen.getByText(copy.imports.columns.levelsHint).textContent ?? "";
        expect(hint).toContain("Root Entity column makes hub entities");
        expect(hint).toContain("WordPress categories");
        expect(hint).toContain("when a page under them is published");
    });

    it("edits the groups and the header row of the open sheet", () => {
        const props = show();
        const [asIndent, asLevel, asNote] = screen.getAllByRole("switch", { name: "Notes" });
        fireEvent.click(asNote ?? document.body);
        expect(props.onOptions).toHaveBeenLastCalledWith({ levelColumns: ["Category"], noteColumns: ["Notes"] });
        fireEvent.click(asLevel ?? document.body);
        expect(props.onOptions).toHaveBeenLastCalledWith({ levelColumns: ["Category", "Notes"], noteColumns: [] });
        fireEvent.click(asIndent ?? document.body);
        expect(props.onOptions).toHaveBeenLastCalledWith({
            levelColumns: ["Category"],
            noteColumns: [],
            indentColumns: ["Notes"],
        });
        fireEvent.click(screen.getByRole("switch", { name: copy.imports.columns.noHeader }));
        expect(props.onHeaderless).toHaveBeenCalledWith(true);
    });

    it("offers the saved mappings only when the site has some", () => {
        show();
        expect(screen.queryByText(copy.imports.columns.saved)).toBeNull();
        held.mappings = [
            { id: "m-1", name: "Client", columns: { path: "URL" }, options: {}, createdAt: null, updatedAt: null },
        ];
        show();
        expect(screen.getAllByText(copy.imports.columns.saved).length).toBeGreaterThan(0);
    });
});
