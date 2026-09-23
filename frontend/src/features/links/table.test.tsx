import { render, screen, within } from "@testing-library/react";
import type { PageAudit } from "../../data/types.js";
import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { auditRows, row } from "./model/fixture.js";
import { AuditTable } from "./table.js";

function table(rows: readonly PageAudit[] = auditRows, narrowed = false) {
    return render(
        <AuditTable
            siteId="s1"
            rows={rows}
            selectedId={null}
            narrowed={narrowed}
            onOpen={() => {}}
            onReset={() => {}}
            onOpenGraph={() => {}}
        />,
    );
}

function rowFor(path: string): HTMLElement {
    const cell = screen.getByText(path);
    const held = cell.closest('[role="row"]');
    if (held === null) {
        throw new Error("no row carries " + path);
    }
    return held as HTMLElement;
}

function dotOf(path: string): Element {
    const marker = rowFor(path).querySelector("span[aria-hidden]");
    if (marker === null) {
        throw new Error("the row for " + path + " carries no severity marker");
    }
    return marker;
}

describe("the audit table", () => {
    it("shows a row for every audited page", () => {
        table();

        for (const audited of auditRows) {
            expect(screen.getByText(audited.path)).toBeDefined();
        }
    });

    it("marks a page missing a required link as danger and one merely missing a link as warn", () => {
        table([
            row({ pageId: "p-owes", path: "/owes/", entityName: "Owes", targets: 4, required: 2, satisfied: 2, missing: 2, missingRequired: 1 }),
            row({ pageId: "p-short", path: "/short/", entityName: "Short", targets: 4, required: 1, satisfied: 3, missing: 1 }),
            row({ pageId: "p-whole", path: "/whole/", entityName: "Whole", targets: 2, required: 1, satisfied: 2 }),
        ]);

        expect(dotOf("/owes/").className).toContain("bg-danger");
        expect(dotOf("/short/").className).toContain("bg-warn");
        expect(dotOf("/whole/").className).toContain("bg-ok");
    });

    it("says how many of the missing links were required", () => {
        table([
            row({ pageId: "p-owes", path: "/owes/", entityName: "Owes", targets: 4, required: 2, satisfied: 2, missing: 3, missingRequired: 2 }),
        ]);

        const held = within(rowFor("/owes/"));
        expect(held.getByText("3")).toBeDefined();
        expect(held.getByText(copy.links.cell.required(2))).toBeDefined();
    });

    it("marks a page whose links the policy blocked as danger even with nothing missing", () => {
        table([row({ pageId: "p-blocked", path: "/blocked/", entityName: "Blocked", targets: 3, required: 1, satisfied: 3, blocked: 1 })]);

        expect(dotOf("/blocked/").className).toContain("bg-danger");
    });

    it("offers the graph when there is nothing to audit and a reset when the filters hid it", () => {
        const first = table([], false);
        expect(screen.getByText(copy.links.empty.title)).toBeDefined();
        expect(screen.getByRole("button", { name: copy.links.empty.graph })).toBeDefined();
        first.unmount();

        table([], true);
        expect(screen.getByText(copy.links.empty.noMatch)).toBeDefined();
        expect(screen.getByRole("button", { name: copy.links.filters.reset })).toBeDefined();
    });
});
