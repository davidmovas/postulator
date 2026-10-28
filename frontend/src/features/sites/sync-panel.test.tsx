import { fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { LatestSync } from "../../data/hooks/sync.js";
import type { SyncProgress } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { aSite, siteSettings } from "../../testing/sites.js";
import { aProgress } from "../../testing/sync.js";

const sent: unknown[] = [];
const retried: unknown[] = [];

const held: { active: SyncProgress | null; latest: LatestSync } = {
    active: null,
    latest: { run: undefined, progress: undefined },
};

vi.mock("../../data/hooks/sync.js", () => ({
    useActiveSync: () => held.active,
    useLatestSync: () => held.latest,
    useSyncSite: () => ({
        mutate: (request: unknown) => {
            sent.push(request);
        },
        isPending: false,
    }),
}));

vi.mock("../../data/hooks/runs.js", () => ({
    useRunItems: (runId: string) => ({
        data: { pages: [{ items: [{ id: "i-old", runId, status: "failed" }], hasMore: false }] },
        isPending: false,
    }),
    useRetryStep: () => ({
        mutate: (request: unknown) => {
            retried.push(request);
        },
        isPending: false,
        error: null,
    }),
    useRegenerate: () => ({ mutate: () => undefined, isPending: false, error: null }),
}));

const { SiteSyncPanel } = await import("./sync-panel.js");

const said = copy.sites.sync;

const failure =
    "reading the body of /about/: WordPress answered 500 on GET /wp-json/wp/v2/pages/12: Internal Server Error (internal_server_error)";

beforeEach(() => {
    sent.length = 0;
    retried.length = 0;
    held.active = null;
    held.latest = { run: undefined, progress: undefined };
});

describe("the site's sync options", () => {
    it("syncs what changed since the last walk when no option is ticked", () => {
        renderScreen(<SiteSyncPanel site={aSite()} />);

        fireEvent.click(screen.getByRole("button", { name: said.start }));

        expect(sent).toStrictEqual([{ siteId: "s1", full: false, renderRescan: false }]);
    });

    it("asks for a full walk", () => {
        renderScreen(<SiteSyncPanel site={aSite()} />);

        fireEvent.click(screen.getByLabelText(said.full));
        fireEvent.click(screen.getByRole("button", { name: said.start }));

        expect(sent).toStrictEqual([{ siteId: "s1", full: true, renderRescan: false }]);
    });

    it("asks for every page to be rendered again, with a full walk or without", () => {
        renderScreen(<SiteSyncPanel site={aSite()} />);

        fireEvent.click(screen.getByLabelText(said.rescan));
        fireEvent.click(screen.getByRole("button", { name: said.start }));
        fireEvent.click(screen.getByLabelText(said.full));
        fireEvent.click(screen.getByRole("button", { name: said.start }));

        expect(sent).toStrictEqual([
            { siteId: "s1", full: false, renderRescan: true },
            { siteId: "s1", full: true, renderRescan: true },
        ]);
    });

    it("says when the site was last walked and never for a site never walked", () => {
        const never = siteSettings({
            observation: { lastStartedAt: null, lastFinishedAt: null, lastFullWalkAt: null, lastModifiedSeen: null, sinceFullWalk: 0 },
        });
        renderScreen(<SiteSyncPanel site={aSite({ settings: never })} />);

        expect(screen.getByText(said.lastWalk)).toBeDefined();
        expect(screen.getAllByText("never")).toHaveLength(2);
    });
});

describe("the site's sync while one is under way", () => {
    it("follows the stage and the pages read, and starts no second one", async () => {
        const user = userEvent.setup();
        held.active = aProgress({ runId: "r-live", stage: "render", pulled: 150 });
        renderScreen(<SiteSyncPanel site={aSite()} />);

        expect(screen.getByText("Visiting the pages · 150 pages read")).toBeDefined();
        const busy = screen.getByRole("button", { name: copy.sync.syncing("visiting the pages") });
        expect(busy).toHaveProperty("disabled", true);
        expect(screen.getByLabelText(said.full)).toHaveProperty("disabled", true);
        expect(screen.getByLabelText(said.rescan)).toHaveProperty("disabled", true);

        await user.click(busy);
        expect(sent).toEqual([]);
    });
});

describe("the site's last sync when it stopped", () => {
    it("shows its error in full and resumes it from the last saved batch", async () => {
        const user = userEvent.setup();
        held.latest = {
            run: undefined,
            progress: aProgress({ runId: "r-old", status: "failed", stage: "content", error: failure }),
        };
        renderScreen(<SiteSyncPanel site={aSite()} />);

        const banner = screen.getByRole("note");
        expect(within(banner).getByText("The sync stopped while reading the pages")).toBeDefined();
        expect(within(banner).getByText(failure)).toBeDefined();

        await user.click(within(banner).getByRole("button", { name: copy.sync.resume }));

        expect(retried).toEqual([{ itemId: "i-old" }]);
    });

    it("says nothing of a last sync that finished", () => {
        held.latest = { run: undefined, progress: aProgress({ runId: "r-old", status: "completed", stage: "done" }) };
        renderScreen(<SiteSyncPanel site={aSite()} />);

        expect(screen.queryByRole("note")).toBeNull();
    });
});
