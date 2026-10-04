import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { ImportCounts, ImportMapping } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import type { StepApplyProps } from "./step-apply.js";
import { StepApply } from "./step-apply.js";

const blank: ImportMapping = { columns: null, options: {}, createdAt: null, updatedAt: null };

function sheetMapping(sheet: string): { sheet: string; mapping: ImportMapping } {
    return { sheet, mapping: { ...blank, options: { sheets: [sheet] } } };
}

const workbook = { mapping: blank, sheets: [sheetMapping("Catalog"), sheetMapping("Compounds")] };
const single = { mapping: { ...blank, options: { sheets: ["Catalog"] } } };

const counts: ImportCounts = {
    entitiesCreated: 4,
    entitiesUpdated: 1,
    edgesCreated: 3,
    pagesCreated: 5,
    pagesUpdated: 0,
    categoriesCreated: 6,
    categoriesDeleted: 2,
    skipped: 0,
};

function show(part: Partial<StepApplyProps> = {}) {
    const props: StepApplyProps = {
        siteId: "s1",
        rows: 15,
        request: workbook,
        applied: null,
        blocked: null,
        saveAs: "",
        counts: null,
        busy: false,
        thrown: null,
        onSaveAs: vi.fn(),
        onBack: vi.fn(),
        onApply: vi.fn(),
        onAgain: vi.fn(),
        ...part,
    };
    renderScreen(<StepApply {...props} />);
    return props;
}

describe("applying a workbook", () => {
    it("says which sheets it reads, in the order of the workbook", () => {
        show();
        expect(screen.getByText(copy.imports.apply.titleWorkbook)).toBeDefined();
        expect(screen.getByText(/Reads Catalog and Compounds, in the order of the workbook\. 15 rows\./)).toBeDefined();
    });

    it("explains that one mapping is saved for each sheet, by the name typed", () => {
        const props = show({ saveAs: "Client" });
        expect(
            screen.getByText(`Optional. One mapping is saved for each sheet: “Client / Catalog”, “Client / Compounds”.`),
        ).toBeDefined();
        fireEvent.change(screen.getByLabelText(copy.imports.apply.saveAs), { target: { value: "Client 2" } });
        expect(props.onSaveAs).toHaveBeenCalledWith("Client 2");
    });

    it("shows the per-sheet names with the placeholder before a name is typed", () => {
        show();
        expect(screen.getByText(/“Client workbook \/ Catalog”/)).toBeDefined();
    });

    it("saves one mapping under the name typed for a single sheet", () => {
        show({ request: single });
        expect(screen.getByText(copy.imports.apply.title)).toBeDefined();
        expect(screen.getByText(copy.imports.apply.saveOne)).toBeDefined();
    });

    it("applies, and refuses to while the preview reports errors", () => {
        const props = show();
        fireEvent.click(screen.getByRole("button", { name: copy.imports.apply.start }));
        expect(props.onApply).toHaveBeenCalledOnce();
        show({ blocked: copy.imports.preview.blocked });
        const shut = screen.getAllByRole("button", { name: copy.imports.apply.start })[1];
        expect(shut?.hasAttribute("disabled")).toBe(true);
    });

    it("says which sheets were applied and which mappings were saved", () => {
        show({ counts, applied: { ...workbook, options: { saveMappingAs: "Client" } } });
        expect(screen.getByText("Applied Catalog and Compounds, in the order of the workbook.")).toBeDefined();
        expect(screen.getByText("Saved 2 mappings: “Client / Catalog”, “Client / Compounds”.")).toBeDefined();
        expect(screen.getByText(copy.imports.apply.pagesCreated)).toBeDefined();
    });

    it("says nothing was saved when no name was given", () => {
        show({ counts, applied: { ...single, options: {} } });
        expect(screen.getByText("Applied Catalog.")).toBeDefined();
        expect(screen.queryByText(/^Saved/)).toBeNull();
    });
});
