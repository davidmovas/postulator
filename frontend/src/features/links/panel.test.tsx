import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { LinkAuditPage, PageAudit } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { auditRows, mugsDetail, plannedDetail, plannedRow, row } from "./model/fixture.js";

const held: { detail: LinkAuditPage | undefined; pending: boolean } = { detail: mugsDetail, pending: false };

vi.mock("../../data/hooks/reports.js", () => ({
    useLinkAuditPage: () => ({ data: held.detail, isPending: held.pending, error: null }),
}));

const { AuditPanel } = await import("./panel.js");

function panel(audited: PageAudit = auditRows[1], onRelink: (pageId: string) => void = () => {}) {
    return renderScreen(<AuditPanel siteId="s1" row={audited} onClose={() => {}} onRelink={onRelink} />);
}

describe("the audit panel", () => {
    it("names the page it audits and the required link it is missing", () => {
        panel();

        expect(screen.getAllByText("/pottery/mugs/").length).toBeGreaterThan(0);
        expect(screen.getByText("/pottery/")).toBeDefined();
        expect(screen.getByText(copy.links.panel.required)).toBeDefined();
    });

    it("keeps the blocked targets apart from the ones the page can still carry", () => {
        panel();

        expect(screen.getByText(copy.links.panel.blocked)).toBeDefined();
        expect(screen.getByText(copy.links.panel.blockedHint)).toBeDefined();
    });

    it("counts the links the graph does not know", () => {
        panel();

        expect(screen.getByText(copy.links.panel.extra)).toBeDefined();
        expect(screen.getByText("https://example.org/")).toBeDefined();
    });

    it("relinks the page it is showing", () => {
        const asked: string[] = [];
        panel(auditRows[1], (pageId) => asked.push(pageId));

        fireEvent.click(screen.getByRole("button", { name: copy.links.relink.page }));
        expect(asked).toStrictEqual(["p-mugs"]);
    });

    it("says a planned page owes its links once it is written", () => {
        held.detail = plannedDetail;
        try {
            panel(plannedRow);

            expect(screen.getByText(copy.links.panel.notWritten)).toBeDefined();
            expect(screen.getAllByText(copy.links.states.awaiting_page ?? "").length).toBe(3);
            expect(screen.queryByText(copy.links.panel.missing)).toBeNull();
            const button = screen.getByRole("button", { name: copy.links.relink.page });
            expect(button.hasAttribute("disabled")).toBe(true);
        } finally {
            held.detail = mugsDetail;
        }
    });

    it("refuses to relink a page the audit skipped", () => {
        panel(row({ pageId: "p-blog", path: "/blog/diary/", status: "exists", skipReason: "unmapped" }));

        const button = screen.getByRole("button", { name: copy.links.relink.page });
        expect(button.hasAttribute("disabled")).toBe(true);
    });
});
