import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { ImportMapping, ImportSheet } from "../../data/types.js";
import type { OptionsPanelProps } from "./options-panel.js";

const held: { mappings: ImportMapping[] } = { mappings: [] };

vi.mock("../../data/hooks/imports.js", () => ({
    useMappings: () => ({ data: { mappings: held.mappings }, isPending: false, error: null }),
}));

const { OptionsPanel } = await import("./options-panel.js");

function sheet(name: string, rows: number, roots: string[] = []): ImportSheet {
    return {
        name,
        rows,
        headers: [],
        detected: { columns: {}, options: roots.length === 0 ? {} : { levelColumns: roots }, createdAt: null, updatedAt: null },
    };
}

const workbook = [sheet("Catalog", 3, ["Root Entity", "Root"]), sheet("Compounds", 12), sheet("Readme", 0)];

function show(part: Partial<OptionsPanelProps> = {}) {
    const props: OptionsPanelProps = {
        siteId: "s1",
        sheets: workbook,
        chosen: ["Catalog", "Compounds"],
        active: "Catalog",
        settings: {
            columns: { path: "URL" },
            options: { levelColumns: ["Root Entity"], noteColumns: [] },
            mappingId: "",
        },
        headers: ["URL", "Root Entity", "Root", "Category", "Notes"],
        onChoose: vi.fn(),
        onOptions: vi.fn(),
        onHeaderless: vi.fn(),
        onSaved: vi.fn(),
        ...part,
    };
    render(<OptionsPanel {...props} />);
    return props;
}

function offered(title: string): string[] {
    const panel = screen.getByRole("heading", { name: title }).closest("section");
    if (panel === null) {
        throw new Error("no panel is titled " + title);
    }
    return within(panel)
        .queryAllByRole("switch")
        .map((choice) => choice.closest("label")?.textContent ?? "");
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

    it("explains that only a Root Entity or Root column makes a group", () => {
        show();
        const hint = screen.getByText(copy.imports.columns.levelsHint).textContent ?? "";
        expect(hint).toContain("Tick the Root Entity or Root columns");
        expect(hint).toContain("under a hub entity");
        expect(hint).toContain("No other column makes a group");
    });

    it("offers as group columns only the root headers detected for the open sheet", () => {
        show();
        expect(offered(copy.imports.columns.levels)).toStrictEqual(["Root Entity", "Root"]);
        expect(offered(copy.imports.columns.notes)).toStrictEqual(["Root", "Category", "Notes"]);
    });

    it("leaves out of the groups a root header a field already reads", () => {
        show({
            settings: { columns: { path: "URL", entity: "Root" }, options: { levelColumns: [] }, mappingId: "" },
        });
        expect(offered(copy.imports.columns.levels)).toStrictEqual(["Root Entity"]);
    });

    it("offers no group column for a sheet whose detector found no root header", () => {
        show({ active: "Compounds", headers: ["URL", "Category", "Subcategory", "Notes"] });
        expect(offered(copy.imports.columns.levels)).toStrictEqual([]);
        expect(screen.getByText(copy.imports.columns.noGroupColumns)).toBeDefined();
    });

    it("edits the groups and the header row of the open sheet", () => {
        const props = show();
        const [asIndent, asGroup, asNote] = screen.getAllByRole("switch", { name: "Root" });
        fireEvent.click(asNote ?? document.body);
        expect(props.onOptions).toHaveBeenLastCalledWith({ levelColumns: ["Root Entity"], noteColumns: ["Root"] });
        fireEvent.click(asGroup ?? document.body);
        expect(props.onOptions).toHaveBeenLastCalledWith({ levelColumns: ["Root Entity", "Root"], noteColumns: [] });
        fireEvent.click(asIndent ?? document.body);
        expect(props.onOptions).toHaveBeenLastCalledWith({
            levelColumns: ["Root Entity"],
            noteColumns: [],
            indentColumns: ["Root"],
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
