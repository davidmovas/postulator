import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { LinkFilters } from "./filters.js";
import { showLabel } from "./labels.js";
import { rows as auditedRows, showCounts } from "./model/audit.js";
import { auditRows } from "./model/fixture.js";
import type { LinksQuery } from "./model/params.js";
import { defaultQuery, narrowed } from "./model/params.js";

function filters(query: LinksQuery, onChange: (next: LinksQuery) => void = () => {}) {
    const counts = showCounts(auditRows);
    const statusCounts = new Map<string, number>();
    for (const audited of auditRows) {
        statusCounts.set(audited.status, (statusCounts.get(audited.status) ?? 0) + 1);
    }
    return render(<LinkFilters query={query} counts={counts} statusCounts={statusCounts} index={null} onChange={onChange} />);
}

describe("the link filters", () => {
    it("counts every page beside the show it belongs to", () => {
        filters(defaultQuery);

        const counts = showCounts(auditRows);
        const all = screen.getByRole("button", { name: new RegExp(showLabel("all")) });
        expect(all.textContent).toContain(String(counts.all));
    });

    it("narrows the table when a show is picked", () => {
        const asked: LinksQuery[] = [];
        filters(defaultQuery, (next) => asked.push(next));

        fireEvent.click(screen.getByRole("button", { name: new RegExp(showLabel("missing")) }));

        expect(asked).toHaveLength(1);
        expect(asked[0].show).toBe("missing");
        expect(narrowed(asked[0])).toBe(true);
        expect(auditedRows(auditRows, asked[0], null).length).toBeLessThan(auditRows.length);
    });

    it("arms the reset only once something is narrowed", () => {
        const wide = filters(defaultQuery);
        expect(screen.getByRole("button", { name: copy.links.filters.reset }).hasAttribute("disabled")).toBe(true);
        wide.unmount();

        filters({ ...defaultQuery, show: "missing" });
        expect(screen.getByRole("button", { name: copy.links.filters.reset }).hasAttribute("disabled")).toBe(false);
    });

    it("puts the narrowing back on a reset and keeps the order that was chosen", () => {
        const asked: LinksQuery[] = [];
        filters({ ...defaultQuery, show: "blocked", sort: "path" }, (next) => asked.push(next));

        fireEvent.click(screen.getByRole("button", { name: copy.links.filters.reset }));

        expect(asked).toStrictEqual([{ ...defaultQuery, sort: "path" }]);
        expect(narrowed(asked[0])).toBe(false);
    });
});
