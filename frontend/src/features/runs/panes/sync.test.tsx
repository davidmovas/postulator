import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { SyncProgress } from "../../../data/types.js";
import { aProgress } from "../../../testing/sync.js";

const held: { progress: SyncProgress | undefined; pending: boolean; asked: unknown[] } = {
    progress: undefined,
    pending: false,
    asked: [],
};

vi.mock("../../../data/hooks/sync.js", () => ({
    useSyncProgress: (siteId: string, runId?: string) => {
        held.asked.push({ siteId, runId });
        return { data: held.progress, isPending: held.pending, isError: false, error: null };
    },
}));

const { SiteSyncPane } = await import("./sync.js");

const said = copy.sync;

const failure =
    "reading the body of /about/: WordPress answered 500 on GET /wp-json/wp/v2/pages/12: Internal Server Error (internal_server_error)";

function stage(words: string): HTMLElement {
    const list = screen.getByRole("list", { name: said.stagesLabel });
    return within(list).getByText(words).closest("li") as HTMLElement;
}

function count(label: string): string {
    const term = screen.getByText(label, { selector: "dt" });
    return term.nextElementSibling?.textContent ?? "";
}

beforeEach(() => {
    held.progress = undefined;
    held.pending = false;
    held.asked.length = 0;
});

describe("a sync run's own pane", () => {
    it("reads the progress of that sync run on its site", () => {
        held.progress = aProgress();
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(held.asked).toContainEqual({ siteId: "s1", runId: "r-sync" });
    });

    it("lists the stages in order, done up to the one under way", () => {
        held.progress = aProgress({ stage: "render", status: "running" });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        const items = within(screen.getByRole("list", { name: said.stagesLabel })).getAllByRole("listitem");
        expect(items.map((item) => item.textContent)).toEqual([
            "Reading the site's setup",
            "Reading the pages",
            "Reading the store",
            "Visiting the pages",
            "Deriving the page tree",
            "Building the topical map",
            "Reading custom fields",
            "Done",
        ]);
        expect(stage("Visiting the pages").getAttribute("aria-current")).toBe("step");
        expect(stage("Reading the pages").getAttribute("data-stage-state")).toBe("done");
        expect(stage("Deriving the page tree").getAttribute("data-stage-state")).toBe("pending");
    });

    it("says whether the walk reads every page or what changed", () => {
        held.progress = aProgress({ full: true });
        const { unmount } = render(<SiteSyncPane siteId="s1" runId="r-sync" />);
        expect(screen.getByText(said.walk.full)).toBeDefined();
        unmount();

        held.progress = aProgress({ full: false });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);
        expect(screen.getByText(said.walk.incremental)).toBeDefined();
    });

    it("counts what the walk read, created, changed, visited and archived, and the batches it saved", () => {
        held.progress = aProgress({
            pulled: 400,
            created: 3,
            updated: 12,
            products: 40,
            rendered: 120,
            unreachable: 2,
            archived: 1,
            batches: 6,
        });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(count(said.counts.pulled)).toBe("400");
        expect(count(said.counts.created)).toBe("3");
        expect(count(said.counts.updated)).toBe("12");
        expect(count(said.counts.products)).toBe("40");
        expect(count(said.counts.rendered)).toBe("120");
        expect(count(said.counts.unreachable)).toBe("2");
        expect(count(said.counts.archived)).toBe("1");
        expect(count(said.counts.batches)).toBe("6");
    });

    it("counts what the topical map built and says why each page it left has no entity, the commonest first", () => {
        held.progress = aProgress({
            stage: "fields",
            entities: 30,
            placed: 27,
            unplaced: 4,
            unplacedBy: { disagrees: 1, unknown_link: 3 },
        });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(count(said.counts.entities)).toBe("30");
        expect(count(said.counts.placed)).toBe("27");
        expect(count(said.counts.unplaced)).toBe("4");
        const reasons = within(screen.getByRole("list", { name: said.left.title })).getAllByRole("listitem");
        expect(reasons.map((reason) => reason.textContent)).toEqual([
            `${said.left.reasons.unknown_link}3`,
            `${said.left.reasons.disagrees}1`,
        ]);
    });

    it("says no reasons while every page that needs an entity has one", () => {
        held.progress = aProgress({ stage: "done", entities: 4, placed: 3 });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(count(said.counts.unplaced)).toBe("0");
        expect(screen.queryByRole("list", { name: said.left.title })).toBeNull();
    });

    it("counts a large walk exactly rather than rounding it", () => {
        held.progress = aProgress({ pulled: 136_100, batches: 2_723 });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(count(said.counts.pulled)).toBe("136,100");
        expect(count(said.counts.batches)).toBe("2,723");
    });

    it("heads the pane with the stage, which the notice above names as stopped", () => {
        held.progress = aProgress({ status: "failed", stage: "content", error: failure });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(screen.getByRole("heading", { name: "Reading the pages" })).toBeDefined();
        expect(screen.queryByText("The sync stopped while reading the pages")).toBeNull();
    });

    it("says the attempt that failed and counts down to the next while it waits to retry", () => {
        held.progress = aProgress({
            status: "waiting",
            attempts: 1,
            wakeAt: new Date(Date.now() + 42_000).toISOString(),
            error: failure,
        });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(screen.getByText(/The first attempt failed; the next starts in 0:4[12]/)).toBeDefined();
        expect(screen.getByText(said.lastError)).toBeDefined();
    });

    it("shows the error of a stopped sync in full, where it can be selected", () => {
        held.progress = aProgress({ status: "failed", stage: "content", error: failure });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(screen.getByText(said.error)).toBeDefined();
        const shown = screen.getByText(failure);
        expect(shown.tagName).toBe("PRE");
        expect(shown.className).toMatch(/select-text/);
        expect(stage("Reading the pages").getAttribute("data-stage-state")).toBe("failed");
    });

    it("says nothing of an error a running sync does not have", () => {
        held.progress = aProgress({ error: "" });
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(screen.queryByText(said.error)).toBeNull();
        expect(screen.queryByText(said.lastError)).toBeNull();
    });

    it("waits for the progress before it says anything of the stages", () => {
        held.pending = true;
        render(<SiteSyncPane siteId="s1" runId="r-sync" />);

        expect(screen.queryByRole("list", { name: said.stagesLabel })).toBeNull();
        expect(screen.getByLabelText(said.loading)).toBeDefined();
    });
});
