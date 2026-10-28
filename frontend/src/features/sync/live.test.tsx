import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { clearToasts, readToasts } from "../../data/toasts.js";
import { renderScreen } from "../../testing/render.js";
import { aProgress } from "../../testing/sync.js";

const calls: { retried: unknown[]; regenerated: unknown[]; listed: unknown[] } = {
    retried: [],
    regenerated: [],
    listed: [],
};

vi.mock("../../data/hooks/runs.js", () => ({
    useRunItems: (runId: string, status: string) => {
        calls.listed.push({ runId, status });
        return { data: { pages: [{ items: [{ id: "i-sync", runId, status: "failed" }], hasMore: false }] }, isPending: false };
    },
    useRetryStep: () => ({
        mutate: (request: unknown, options?: { onSuccess?: () => void }) => {
            calls.retried.push(request);
            options?.onSuccess?.();
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

const { SyncLive, SyncStopped } = await import("./live.js");

const said = copy.sync;

const error =
    "reading the body of /about/: WordPress answered 500 on GET /wp-json/wp/v2/pages/12: Internal Server Error (internal_server_error)";

beforeEach(() => {
    calls.retried.length = 0;
    calls.regenerated.length = 0;
    calls.listed.length = 0;
    clearToasts();
});

describe("a live sync on a panel", () => {
    it("says the stage and the pages read so far", () => {
        renderScreen(<SyncLive progress={aProgress({ stage: "render", pulled: 150 })} />);

        expect(screen.getByText("Visiting the pages · 150 pages read")).toBeDefined();
    });

    it("marks how far through its stages the walk is", () => {
        renderScreen(<SyncLive progress={aProgress({ stage: "render" })} />);

        const meter = screen.getByRole("img", { name: said.stagesLabel });
        expect(meter.querySelectorAll("[data-stage-state='done']")).toHaveLength(3);
        expect(meter.querySelectorAll("[data-stage-state='current']")).toHaveLength(1);
    });
});

describe("a sync that stopped", () => {
    it("names the stage it stopped in and shows the error in full", () => {
        renderScreen(<SyncStopped runId="r-sync" stage="content" error={error} />);

        const banner = screen.getByRole("note");
        expect(within(banner).getByText("The sync stopped while reading the pages")).toBeDefined();
        expect(within(banner).getByText(error)).toBeDefined();
    });

    it("resumes the failed item from its last saved batch", async () => {
        const user = userEvent.setup();
        renderScreen(<SyncStopped runId="r-sync" stage="content" error={error} />);

        await user.click(screen.getByRole("button", { name: said.resume }));

        expect(calls.listed).toContainEqual({ runId: "r-sync", status: "failed" });
        expect(calls.retried).toEqual([{ itemId: "i-sync" }]);
        expect(readToasts().map((toast) => toast.message)).toEqual([said.resumed]);
    });

    it("starts over only where it is offered, and never in place of resuming", async () => {
        const user = userEvent.setup();
        renderScreen(<SyncStopped runId="r-sync" stage="render" error={error} startOver={true} />);

        await user.click(screen.getByRole("button", { name: said.startOver }));

        expect(calls.regenerated).toEqual([{ runId: "r-sync", itemIds: ["i-sync"] }]);
        expect(calls.retried).toEqual([]);
    });

    it("offers no start over unless asked", () => {
        renderScreen(<SyncStopped runId="r-sync" stage="render" error={error} />);

        expect(screen.queryByRole("button", { name: said.startOver })).toBeNull();
    });

    it("opens the sync when the panel offers it", async () => {
        const user = userEvent.setup();
        const opened: string[] = [];
        renderScreen(
            <SyncStopped
                runId="r-sync"
                stage="render"
                error={error}
                onOpen={(runId) => {
                    opened.push(runId);
                }}
            />,
        );

        await user.click(screen.getByRole("button", { name: said.open }));

        expect(opened).toEqual(["r-sync"]);
    });
});
