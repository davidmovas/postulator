import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { SyncProgress } from "../../data/types.js";
import { aProgress } from "../../testing/sync.js";

const empty = {
    siteId: "s1",
    entities: { total: 0, withCanonicalPage: 0, withPublishedPage: 0 },
    pages: { byStatus: {}, total: 0, unmapped: 0, orphans: 0 },
    edges: { approved: 0, realized: 0 },
    depth: [],
    top: [],
};

const held: { active: SyncProgress | null; sorts: unknown[]; sent: unknown[] } = { active: null, sorts: [], sent: [] };

const loaded = { pages: [{ items: [], hasMore: false }] };

vi.mock("../../data/hooks/browser.js", () => ({
    useOpenExternal: () => ({ mutate: () => undefined, isPending: false }),
}));
vi.mock("../../data/hooks/graph.js", () => ({
    useEdges: () => ({ data: loaded }),
}));
vi.mock("../../data/hooks/pages.js", () => ({
    usePageTree: () => ({ data: { roots: [] } }),
}));
vi.mock("../../data/hooks/reports.js", () => ({
    useSiteOverview: () => ({ data: empty, isPending: false, isError: false }),
}));
vi.mock("../../data/hooks/runs.js", () => ({
    useRuns: (_filter: unknown, sort: unknown) => {
        held.sorts.push(sort);
        return { data: loaded };
    },
}));
vi.mock("../../data/hooks/sites.js", () => ({
    useSite: () => ({ data: { site: { id: "s1", baseUrl: "https://shop.test" } } }),
}));
vi.mock("../../data/hooks/sync.js", () => ({
    usePluginState: () => ({ data: { plugin: { installed: true } } }),
    useActiveSync: () => held.active,
    useSyncSite: () => ({
        mutate: (request: unknown) => {
            held.sent.push(request);
        },
        isPending: false,
    }),
}));
vi.mock("./readiness.js", () => ({ useReadiness: () => null }));
vi.mock("./checklist.js", () => ({ ReadinessChecklist: () => null }));
vi.mock("./observation.js", () => ({ ObservationPanel: () => null }));

const { OverviewScreen } = await import("./screen.js");

function shown(): void {
    render(
        <MemoryRouter initialEntries={["/s/s1/overview"]}>
            <Routes>
                <Route path="/s/:siteId/overview" element={<OverviewScreen />} />
            </Routes>
        </MemoryRouter>,
    );
}

beforeEach(() => {
    held.active = null;
    held.sorts.length = 0;
    held.sent.length = 0;
});

describe("the overview's latest run", () => {
    it("reads the site's runs newest first, so the latest run is the one shown", () => {
        shown();

        expect(held.sorts).toContainEqual({ field: "createdAt", desc: true });
    });
});

describe("the overview's sync", () => {
    it("starts a sync from a site nothing was read from yet", async () => {
        const user = userEvent.setup();
        shown();

        expect(screen.getByText(copy.overview.empty)).toBeDefined();
        const buttons = screen.getAllByRole("button", { name: copy.overview.sync });
        expect(buttons).toHaveLength(2);
        await user.click(buttons[1] as HTMLElement);

        expect(held.sent).toStrictEqual([{ siteId: "s1" }]);
    });

    it("says the stage of the sync under way wherever it offered one, and opens it", () => {
        held.active = aProgress({ runId: "r-live", stage: "inventory" });
        shown();

        const busy = screen.getAllByRole("button", { name: copy.sync.syncing("reading the site's setup") });
        expect(busy.length).toBeGreaterThanOrEqual(2);
        for (const button of busy) {
            expect(button).toHaveProperty("disabled", true);
        }
        expect(screen.queryByRole("button", { name: copy.overview.sync })).toBeNull();
        expect(screen.getAllByRole("button", { name: copy.sync.open }).length).toBeGreaterThanOrEqual(2);
    });
});
