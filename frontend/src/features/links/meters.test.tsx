import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import type { LinkTotals } from "../../data/types.js";
import { AuditStrip } from "./meters.js";
import { typed } from "./model/fixture.js";

const totals: LinkTotals = {
    pages: 16,
    audited: 12,
    targets: 20,
    required: 8,
    satisfied: 6,
    missing: 14,
    missingRequired: 4,
    blocked: 0,
    offGraph: 2,
    orphans: 1,
    pending: 0,
    unpublished: 0,
    byType: typed({ breadcrumb: 40, listing: 18, navigation: 120, contextual: 6, unknown: 2 }),
};

describe("the audit strip", () => {
    it("counts the site's links by type, in words", () => {
        render(<AuditStrip totals={totals} />);

        const strip = within(screen.getByRole("list", { name: copy.links.meters.byType }));
        for (const [label, count] of [
            ["Breadcrumb", "40"],
            ["Listing", "18"],
            ["Navigation", "120"],
            ["Contextual", "6"],
            ["Unclassified", "2"],
        ] as const) {
            const item = strip.getByText(label).closest("li");
            expect(item?.textContent).toBe(`${count}${label}`);
        }
    });

    it("says where the rules that decide what a page owes are kept", () => {
        const { container } = render(<AuditStrip totals={totals} />);
        expect(container.querySelector(`[title="${copy.links.meters.rulesFromTypes}"]`)).not.toBeNull();
        expect(screen.queryByText(/policy/i)).toBeNull();
    });

    it("says in one sentence that only contextual links settle what the graph owes", () => {
        render(<AuditStrip totals={totals} />);

        const strip = within(screen.getByRole("list", { name: copy.links.meters.byType }));
        expect(strip.getByText("Contextual").closest("li")?.getAttribute("title")).toBe(copy.links.meters.contextualOnly);
        expect(strip.getByText("Navigation").closest("li")?.hasAttribute("title")).toBe(false);
    });
});
