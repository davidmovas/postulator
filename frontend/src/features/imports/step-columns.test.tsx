import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { StepColumnsProps } from "./step-columns.js";
import { StepColumns } from "./step-columns.js";

function show(part: Partial<StepColumnsProps> = {}) {
    const props: StepColumnsProps = {
        tabs: [
            { name: "Catalog", rows: 3 },
            { name: "Compounds", rows: 12 },
        ],
        active: "Catalog",
        headers: ["URL", "Title", "Notes"],
        sample: [["/peptides/", "Peptides", "keep it short"]],
        busy: false,
        failure: null,
        columns: { path: "URL", title: "Title" },
        detected: { path: "URL" },
        notice: { kind: "ready" },
        onActivate: vi.fn(),
        onAssign: vi.fn(),
        onBack: vi.fn(),
        onNext: vi.fn(),
        ...part,
    };
    render(<StepColumns {...props} />);
    return props;
}

function next(): HTMLElement {
    return screen.getByRole("button", { name: copy.imports.next });
}

describe("the columns step of a workbook", () => {
    it("puts a tab over each chosen sheet, with its rows, and opens the one clicked", () => {
        const props = show();
        expect(screen.getByRole("tab", { name: /Catalog/ }).getAttribute("aria-selected")).toBe("true");
        expect(screen.getByRole("tab", { name: /Compounds/ }).textContent).toContain("12");
        fireEvent.click(screen.getByRole("tab", { name: /Compounds/ }));
        expect(props.onActivate).toHaveBeenCalledWith("Compounds");
    });

    it("shows no tabs for a file with one sheet", () => {
        show({ tabs: [] });
        expect(screen.queryByRole("tablist")).toBeNull();
    });

    it("shows the open sheet's columns, their samples and what was matched automatically", () => {
        show();
        expect(screen.getByText("keep it short")).toBeDefined();
        expect(screen.getAllByLabelText(copy.imports.columns.detected)).toHaveLength(1);
        expect(screen.getByText(copy.imports.columns.unmapped(1))).toBeDefined();
    });

    it("goes on when every chosen sheet can be read", () => {
        const props = show();
        fireEvent.click(next());
        expect(props.onNext).toHaveBeenCalledOnce();
    });

    it("names the sheet that still lacks a path or an entity and opens it", () => {
        const props = show({ notice: { kind: "otherSheet", sheet: "Compounds" } });
        expect(screen.getByText(copy.imports.columns.otherSheet("Compounds"))).toBeDefined();
        expect(next().hasAttribute("disabled")).toBe(true);
        fireEvent.click(screen.getByRole("button", { name: copy.imports.columns.openSheet("Compounds") }));
        expect(props.onActivate).toHaveBeenCalledWith("Compounds");
    });

    it.each([
        { kind: "needsTarget" as const, text: copy.imports.columns.needsTarget },
        { kind: "unmatched" as const, text: copy.imports.columns.noHeaderHelp },
    ])("stays shut and says why when the open sheet is $kind", ({ kind, text }) => {
        show({ notice: { kind } });
        expect(screen.getByText(text)).toBeDefined();
        expect(next().hasAttribute("disabled")).toBe(true);
    });

    it("asks for a sheet when every sheet is turned off", () => {
        show({ tabs: [], active: "", headers: [], notice: { kind: "noSheet" } });
        expect(screen.getByText(copy.imports.columns.noSheet)).toBeDefined();
        expect(next().hasAttribute("disabled")).toBe(true);
    });

    it("waits for a sheet read without a header row", () => {
        show({ headers: [], sample: [], busy: true });
        expect(screen.getByLabelText(copy.imports.file.reading)).toBeDefined();
    });
});
