import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { RunEventsState } from "../../data/runs/log.js";
import type { Run } from "../../data/types.js";
import { aRun, aSyncRun } from "../../testing/runs.js";
import { aProgress } from "../../testing/sync.js";
import { publishingLabel } from "./labels.js";

const held: { run: Run; asked: unknown[] } = { run: aSyncRun(), asked: [] };

const events: RunEventsState = {
    events: [],
    contiguousSeq: 0,
    maxSeq: 0,
    phase: "live",
    error: null,
    terminal: false,
    version: 0,
};

const idle = { mutate: () => undefined, isPending: false, error: null };

vi.mock("../../data/hooks/runs.js", () => ({
    useRun: () => ({ data: { run: held.run }, isPending: false }),
    useRunItems: () => ({
        data: { pages: [{ items: [], hasMore: false }] },
        isPending: false,
        hasNextPage: false,
        isFetchingNextPage: false,
        fetchNextPage: () => undefined,
    }),
    useRunProgress: () => ({
        run: held.run,
        items: [],
        events,
        stats: held.run.stats,
        progress: new Map(),
        terminal: false,
        gap: false,
    }),
    useCancelRun: () => idle,
    usePauseRun: () => idle,
    useResumeRun: () => idle,
    useRevertRun: () => idle,
    useRegenerate: () => idle,
    useRetryStep: () => idle,
}));

vi.mock("../../data/hooks/templates.js", () => ({
    useTemplate: () => ({ data: undefined, isPending: false, error: null }),
}));

vi.mock("../../data/hooks/sync.js", () => ({
    useSyncProgress: (siteId: string, runId?: string) => {
        held.asked.push({ siteId, runId });
        return { data: aProgress({ runId: "r-sync", stage: "render" }), isPending: false };
    },
}));

vi.mock("./page-index.js", () => ({
    usePageIndex: () => ({ byId: new Map(), complete: true, loading: false }),
    targetPathOf: () => "The whole site",
}));

vi.mock("./start.js", () => ({ StartRunDrawer: () => null }));

const { RunDetailScreen } = await import("./detail.js");

function shown(run: Run): void {
    held.run = run;
    render(
        <MemoryRouter initialEntries={[`/s/s1/runs/${run.id}`]}>
            <Routes>
                <Route path="/s/:siteId/runs/:runId" element={<RunDetailScreen />} />
            </Routes>
        </MemoryRouter>,
    );
}

beforeEach(() => {
    held.asked.length = 0;
});

describe("a run's screen", () => {
    it("shows a sync's stages, counts and walk above its item", () => {
        shown(aSyncRun({ status: "running" }));

        expect(held.asked).toContainEqual({ siteId: "s1", runId: "r-sync" });
        expect(screen.getByRole("region", { name: copy.sync.pane })).toBeDefined();
        expect(screen.getByRole("list", { name: copy.sync.stagesLabel })).toBeDefined();
    });

    it("shows no sync pane for a run over pages", () => {
        shown(aRun({ status: "running" }));

        expect(screen.queryByRole("region", { name: copy.sync.pane })).toBeNull();
        expect(held.asked).toEqual([]);
    });

    it("leaves out of a sync's header what only a run over pages has", () => {
        shown(aSyncRun({ status: "failed" }));

        expect(screen.queryByText(publishingLabel(held.run.publishMode))).toBeNull();
        expect(screen.queryByText(copy.runs.detail.noTemplate)).toBeNull();
        expect(screen.queryByRole("button", { name: copy.runs.revert })).toBeNull();
        expect(screen.getByText(copy.runs.wholeSite)).toBeDefined();
    });

    it("keeps a page run's publishing, template, revert and target count", () => {
        shown(aRun({ status: "completed", templateId: "", targets: ["p1", "p2"] }));

        expect(screen.getByText(publishingLabel(held.run.publishMode))).toBeDefined();
        expect(screen.getByText(copy.runs.detail.noTemplate)).toBeDefined();
        expect(screen.getByRole("button", { name: copy.runs.revert })).toBeDefined();
        expect(screen.getByText(copy.runs.detail.targets(2))).toBeDefined();
    });
});
