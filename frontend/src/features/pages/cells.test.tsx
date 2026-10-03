import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { aPage } from "../../testing/pages.js";
import { DenseTable, TableCell } from "../../ui/index.js";
import { DriftCell, EntityCell, PageTableRow, SyncedCell } from "./cells.js";

function inTable(node: ReactNode) {
    return render(
        <DenseTable columns="1fr 1fr" label="pages">
            {node}
        </DenseTable>,
    );
}

describe("PageTableRow", () => {
    it("names the page it shows and selects or opens it", () => {
        const onSelect = vi.fn();
        const onOpen = vi.fn();
        inTable(
            <PageTableRow page={aPage("p1", "/mugs/", { status: "published" })} selected={true} onSelect={onSelect} onOpen={onOpen}>
                <TableCell>/mugs/</TableCell>
            </PageTableRow>,
        );
        const row = screen.getByRole("row");
        expect(row.dataset["pageRow"]).toBe("true");
        expect(row.dataset["pageId"]).toBe("p1");
        expect(row.dataset["pageStatus"]).toBe("published");
        expect(row.getAttribute("aria-selected")).toBe("true");
        expect(row.tabIndex).toBe(0);

        fireEvent.click(row);
        expect(onSelect).toHaveBeenCalledWith("p1");
        expect(onOpen).not.toHaveBeenCalled();

        fireEvent.doubleClick(row);
        expect(onOpen).toHaveBeenCalledTimes(1);

        fireEvent.keyDown(row, { key: "Enter" });
        expect(onOpen).toHaveBeenCalledTimes(2);
        expect(onOpen).toHaveBeenLastCalledWith("p1");

        fireEvent.keyDown(row, { key: "Space" });
        expect(onOpen).toHaveBeenCalledTimes(2);
    });
});

describe("EntityCell", () => {
    it("says a page has no entity", () => {
        inTable(<EntityCell entityId={null} entityName={null} />);
        const cell = screen.getByRole("cell");
        expect(cell.textContent).toBe(copy.pages.unmapped);
        expect(cell.className).toContain("text-ink-dim");
        expect(cell.querySelector("svg")).not.toBeNull();
    });

    it("names the entity a page carries, or says it is mapped while the name loads", () => {
        const { rerender } = inTable(<EntityCell entityId="e1" entityName="Ceramic Mugs" />);
        expect(screen.getByRole("cell").textContent).toBe("Ceramic Mugs");
        expect(screen.getByRole("cell").className).toContain("text-ink");
        expect(screen.getByRole("cell").className).not.toContain("text-ink-dim");
        rerender(
            <DenseTable columns="1fr" label="pages">
                <EntityCell entityId="e1" entityName={null} />
            </DenseTable>,
        );
        expect(screen.getByRole("cell").textContent).toBe(copy.pages.mapped);
    });
});

describe("DriftCell", () => {
    it("marks a drifted page with a warning a screen reader can read", () => {
        inTable(<DriftCell drift={true} />);
        const cell = screen.getByRole("cell");
        expect(cell.querySelector(`[title="${copy.pages.drift.title}"]`)).not.toBeNull();
        expect(cell.querySelector(".sr-only")?.textContent).toBe(copy.pages.drift.badge);
    });

    it("stays empty for a page that did not drift", () => {
        inTable(<DriftCell drift={false} />);
        expect(screen.getByRole("cell").childElementCount).toBe(0);
    });
});

describe("SyncedCell", () => {
    it.each([["2026-09-24T09:00:00Z"], [null]])("says when the page was last synced (%s)", (at) => {
        inTable(<SyncedCell at={at} />);
        const cell = screen.getByRole("cell");
        expect(cell.textContent).toBe(relativeTime(at));
        expect(cell.getAttribute("title")).toBe(absoluteTime(at));
    });
});
