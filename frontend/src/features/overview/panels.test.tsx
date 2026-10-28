import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { Run, SyncProgress } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { aRun, aSyncRun } from "../../testing/runs.js";
import { aProgress } from "../../testing/sync.js";
import { runSummary } from "./model/overview.js";

const held: { progress: SyncProgress | undefined; asked: unknown[]; retried: unknown[] } = {
    progress: undefined,
    asked: [],
    retried: [],
};

vi.mock("../../data/hooks/sync.js", () => ({
    useSyncProgress: (siteId: string, runId?: string) => {
        held.asked.push({ siteId, runId });
        return { data: held.progress };
    },
}));

vi.mock("../../data/hooks/runs.js", () => ({
    useRunItems: (runId: string) => ({
        data: { pages: [{ items: [{ id: "i-sync", runId, status: "failed" }], hasMore: false }] },
        isPending: false,
    }),
    useRetryStep: () => ({
        mutate: (request: unknown) => {
            held.retried.push(request);
        },
        isPending: false,
        error: null,
    }),
    useRegenerate: () => ({ mutate: () => undefined, isPending: false, error: null }),
}));

const { RunPanel } = await import("./panels.js");

const failure = "the step ran past its time limit of 10 minutes and was stopped; a retry starts from the last point it saved";

function shown(run: Run): void {
    renderScreen(<RunPanel summary={runSummary([run])} siteId="s1" onStart={() => undefined} />);
}

beforeEach(() => {
    held.progress = undefined;
    held.asked.length = 0;
    held.retried.length = 0;
});

describe("the latest run panel with a sync under way", () => {
    it("follows the stage and the pages read instead of counting one item", () => {
        held.progress = aProgress({ runId: "r-sync", stage: "render", pulled: 150 });
        shown(aSyncRun({ status: "running" }));

        expect(held.asked).toContainEqual({ siteId: "s1", runId: "r-sync" });
        expect(screen.getByText("Visiting the pages · 150 pages read")).toBeDefined();
        expect(screen.queryByText(copy.overview.run.items(0, 1))).toBeNull();
        expect(screen.getByRole("link", { name: /Sync/ }).getAttribute("href")).toBe("/s/s1/runs/r-sync");
    });
});

describe("the latest run panel with a sync that stopped", () => {
    it("shows the error in full and resumes the sync from its last saved batch", async () => {
        const user = userEvent.setup();
        held.progress = aProgress({ runId: "r-sync", status: "failed", stage: "render", pulled: 150, error: failure });
        shown(aSyncRun({ status: "failed", error: failure }));

        const banner = screen.getByRole("note");
        expect(within(banner).getByText("The sync stopped while visiting the pages")).toBeDefined();
        expect(within(banner).getByText(failure)).toBeDefined();

        await user.click(within(banner).getByRole("button", { name: copy.sync.resume }));

        expect(held.retried).toEqual([{ itemId: "i-sync" }]);
    });
});

describe("the latest run panel with a run over pages", () => {
    it("counts the run's items and reads no sync", () => {
        shown(aRun({ status: "running" }));

        expect(screen.getByText(copy.overview.run.items(4, 12))).toBeDefined();
        expect(held.asked).toEqual([]);
    });
});
