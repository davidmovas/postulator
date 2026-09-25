import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { LinkAudit, PageAudit } from "../../data/types.js";
import { auditRows, row } from "./model/fixture.js";
import { relinkCap } from "./model/relink.js";

const held: { audit: LinkAudit | undefined } = { audit: undefined };

vi.mock("../../data/hooks/reports.js", () => ({
    useLinkAudit: () => ({ data: held.audit, isPending: held.audit === undefined, error: null }),
    useLinkAuditPage: () => ({ data: undefined, isPending: false, error: null }),
}));

vi.mock("../../data/hooks/graph.js", () => ({
    useGraph: () => ({ data: { entities: [], edges: [] }, isPending: false, error: null }),
}));

vi.mock("../runs/start.js", () => ({
    StartRunDrawer: () => null,
}));

const { LinksScreen } = await import("./screen.js");

function audited(rows: readonly PageAudit[]): LinkAudit {
    return {
        siteId: "s1",
        policy: { id: "p1", name: "Default", forbidExternal: false, forbidSelf: true, anchorStrategy: "first" },
        totals: {
            pages: rows.length,
            audited: rows.length,
            targets: 0,
            required: 0,
            satisfied: 0,
            missing: 0,
            missingRequired: 0,
            blocked: 0,
            offGraph: 0,
            orphans: 0,
            pending: 0,
            unpublished: 0,
        },
        pages: [...rows],
    };
}

function links(rows: readonly PageAudit[]) {
    held.audit = audited(rows);
    return render(
        <MemoryRouter initialEntries={["/s/s1/links"]}>
            <Routes>
                <Route path="/s/:siteId/links" element={<LinksScreen />} />
            </Routes>
        </MemoryRouter>,
    );
}

function owing(count: number): PageAudit[] {
    const rows: PageAudit[] = [];
    for (let index = 0; index < count; index += 1) {
        rows.push(row({ pageId: `p${index}`, path: `/p${index}/`, entityName: `P${index}`, targets: 2, required: 1, missing: 1, missingRequired: 1 }));
    }
    return rows;
}

describe("the linking screen's relink action", () => {
    it("counts the pages that owe a link", () => {
        links(owing(3));

        expect(screen.getByRole("button", { name: copy.links.relink.start(3) })).toBeDefined();
    });

    it("says one page in the singular", () => {
        links(owing(1));

        expect(screen.getByRole("button", { name: copy.links.relink.start(1) })).toBeDefined();
    });

    it("stands down when every audited page carries what the graph asks for", () => {
        links([row({ pageId: "p-whole", path: "/whole/", entityName: "Whole", targets: 2, required: 1, satisfied: 2 })]);

        const button = screen.getByRole("button", { name: copy.links.relink.none });
        expect(button.hasAttribute("disabled")).toBe(true);
        expect(button.getAttribute("title")).toBe(copy.links.relink.noneTitle);
    });

    it("caps the run at what a run can take and says so", () => {
        links(owing(relinkCap + 7));

        const button = screen.getByRole("button", { name: copy.links.relink.start(relinkCap) });
        expect(button.hasAttribute("disabled")).toBe(false);
        expect(button.getAttribute("title")).toContain(copy.links.relink.capped(relinkCap));
    });

    it("skips the pages the audit could not read", () => {
        links([...owing(2), ...auditRows.filter((audit) => audit.skipReason !== "")]);

        expect(screen.getByRole("button", { name: copy.links.relink.start(2) })).toBeDefined();
    });
});
