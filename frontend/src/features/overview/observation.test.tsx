import { screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { ModelSummary } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";

const replica: ModelSummary = {
    byType: { page: 9, product: 4, product_cat: 3 },
    roots: { found: 8, empty_raw: 1, dynamic_only: 1, archive: 3, not_found: 0 },
    linksByType: { breadcrumb: 40, listing: 18, navigation: 120, contextual: 6, unknown: 2 },
    technical: { cart: 1, checkout: 1, account: 1, shop: 1 },
    findings: { parent_unresolved: 3, parent_ambiguous: 1, render_redirected: 1 },
    trailsByMethod: { href: 6, redirect: 1, name: 2 },
    trailsByReason: { jsonld_breadcrumb: 7, aria_breadcrumb: 2 },
    pages: 16,
    observed: 15,
    rendered: 13,
    trailsResolved: 9,
    listings: 4,
    readOnly: 3,
    lastObservedAt: "2026-10-18T07:00:00Z",
    lastFullWalkAt: "2026-10-12T07:00:00Z",
};

const held: { summary: ModelSummary | undefined } = { summary: replica };

vi.mock("../../data/hooks/pages.js", () => ({
    useModelSummary: () => ({ data: held.summary, isPending: held.summary === undefined, error: null }),
}));

const { ObservationPanel, countRows } = await import("./observation.js");

const said = copy.overview.observation;

function group(title: string) {
    return within(screen.getByRole("region", { name: title }));
}

describe("the observation summary", () => {
    it("counts the pages observed and rendered", () => {
        held.summary = replica;
        renderScreen(<ObservationPanel siteId="s1" />);

        expect(screen.getByText(said.observedOf(15, 16))).toBeDefined();
        expect(screen.getByText(said.rendered)).toBeDefined();
        expect(screen.getByText(said.listings)).toBeDefined();
    });

    it("words every count it groups and leaves the empty ones out", () => {
        held.summary = replica;
        renderScreen(<ObservationPanel siteId="s1" />);

        expect(group(said.groups.roots).getByText("Archive page")).toBeDefined();
        expect(group(said.groups.roots).queryByText("Content not located")).toBeNull();
        expect(group(said.groups.trailsByMethod).getByText("Through a redirect")).toBeDefined();
        expect(group(said.groups.trailsByReason).getByText("JSON-LD breadcrumb")).toBeDefined();
        expect(group(said.groups.linksByType).getByText("Contextual")).toBeDefined();
        expect(group(said.groups.technical).getByText("Checkout")).toBeDefined();
        expect(group(said.groups.findings).getByText("Crumb without a page")).toBeDefined();
    });

    it("prints no stored key", () => {
        held.summary = replica;
        const { container } = renderScreen(<ObservationPanel siteId="s1" />);

        for (const raw of ["empty_raw", "dynamic_only", "jsonld_breadcrumb", "parent_unresolved", "render_redirected", "href"]) {
            expect(container.textContent).not.toContain(raw);
        }
    });

    it("says a site never observed is not in the model yet", () => {
        held.summary = { ...replica, observed: 0, rendered: 0, lastObservedAt: null, lastFullWalkAt: null };
        renderScreen(<ObservationPanel siteId="s1" />);

        expect(screen.getByText(said.none)).toBeDefined();
        expect(screen.getByText(said.walks("never", "never"))).toBeDefined();
    });
});

describe("the summary's rows", () => {
    it("put the largest count first and the ties in order of their words", () => {
        expect(countRows({ b: 2, a: 2, c: 5, d: 0 }, (key) => key.toUpperCase())).toStrictEqual([
            { key: "c", label: "C", count: 5 },
            { key: "a", label: "A", count: 2 },
            { key: "b", label: "B", count: 2 },
        ]);
        expect(countRows(null, (key) => key)).toStrictEqual([]);
    });
});
