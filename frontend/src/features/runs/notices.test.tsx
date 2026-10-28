import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { RunEventsState } from "../../data/runs/log.js";
import type { Run } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { aRun, aSyncRun } from "../../testing/runs.js";
import { aProgress } from "../../testing/sync.js";
import { runView } from "./authority.js";

const calls: { retried: unknown[]; regenerated: unknown[] } = { retried: [], regenerated: [] };

vi.mock("../../data/hooks/runs.js", () => ({
    useRunItems: (runId: string) => ({
        data: { pages: [{ items: [{ id: `${runId}-item`, runId, status: "failed" }], hasMore: false }] },
        isPending: false,
    }),
    useRetryStep: () => ({
        mutate: (request: unknown) => {
            calls.retried.push(request);
        },
        isPending: false,
        error: null,
    }),
    useRegenerate: () => ({
        mutate: (request: unknown) => {
            calls.regenerated.push(request);
        },
        isPending: false,
        error: null,
    }),
}));

vi.mock("../../data/hooks/sync.js", () => ({
    useSyncProgress: () => ({ data: aProgress({ runId: "r-sync", status: "failed", stage: "render" }) }),
}));

const { RunNotices } = await import("./notices.js");

const events: RunEventsState = {
    events: [],
    contiguousSeq: 0,
    maxSeq: 0,
    phase: "live",
    error: null,
    terminal: true,
    version: 0,
};

const failure = "the step ran past its time limit of 10 minutes and was stopped; a retry starts from the last point it saved";

function shown(run: Run, failedItems: number): void {
    renderScreen(
        <RunNotices
            run={run}
            view={runView(run)}
            events={events}
            gap={false}
            failedItems={failedItems}
            heldItems={0}
            notedItems={0}
        />,
    );
}

beforeEach(() => {
    calls.retried.length = 0;
    calls.regenerated.length = 0;
});

describe("the notice of a sync that stopped", () => {
    it("offers to resume the sync and never to regenerate a page", () => {
        shown(aSyncRun({ status: "failed", error: failure, budget: { maxUsd: 1, maxTokens: 0 } }), 1);

        expect(screen.queryByRole("button", { name: copy.runs.regenerateFailed(1) })).toBeNull();
        const banner = screen.getByText("The sync stopped while visiting the pages").closest("[role='note']") as HTMLElement;
        expect(within(banner).getByRole("button", { name: copy.sync.resume })).toBeDefined();
        expect(within(banner).getByText(copy.sync.stopped.body)).toBeDefined();
    });

    it("resumes the failed item from the last batch it saved", async () => {
        const user = userEvent.setup();
        shown(aSyncRun({ status: "failed", error: failure, budget: { maxUsd: 1, maxTokens: 0 } }), 1);

        await user.click(screen.getByRole("button", { name: copy.sync.resume }));

        expect(calls.retried).toEqual([{ itemId: "r-sync-item" }]);
        expect(calls.regenerated).toEqual([]);
    });

    it("starts the walk over only when asked", async () => {
        const user = userEvent.setup();
        shown(aSyncRun({ status: "failed", error: failure, budget: { maxUsd: 1, maxTokens: 0 } }), 1);

        await user.click(screen.getByRole("button", { name: copy.sync.startOver }));

        expect(calls.regenerated).toEqual([{ runId: "r-sync", itemIds: ["r-sync-item"] }]);
        expect(calls.retried).toEqual([]);
    });

    it("leaves the reason to the sync's own pane rather than saying it twice", () => {
        shown(aSyncRun({ status: "failed", error: failure, budget: { maxUsd: 1, maxTokens: 0 } }), 1);

        expect(screen.queryByText(copy.runs.detail.failure)).toBeNull();
        expect(screen.queryByText(failure)).toBeNull();
    });
});

describe("the notice of a page run that failed", () => {
    it("still regenerates its failed pages and says why the run failed", () => {
        shown(aRun({ status: "failed", error: failure }), 1);

        expect(screen.getByRole("button", { name: copy.runs.regenerateFailed(1) })).toBeDefined();
        expect(screen.queryByRole("button", { name: copy.sync.resume })).toBeNull();
        expect(screen.getByText(copy.runs.detail.failure)).toBeDefined();
        expect(screen.getByText(failure)).toBeDefined();
    });
});
