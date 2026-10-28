import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { SyncProgress } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { aProgress } from "../../testing/sync.js";

const held: { active: SyncProgress | null; sent: unknown[] } = { active: null, sent: [] };

vi.mock("../../data/hooks/sync.js", () => ({
    useActiveSync: () => held.active,
    useSyncSite: () => ({
        mutate: (request: unknown, options?: { onSuccess?: (answered: { runId: string }) => void }) => {
            held.sent.push(request);
            options?.onSuccess?.({ runId: "r-new" });
        },
        isPending: false,
    }),
}));

const { DriftTable } = await import("./tables.js");

const said = copy.reports.drift;

beforeEach(() => {
    held.active = null;
    held.sent.length = 0;
});

describe("the drift table's re-sync", () => {
    it("starts a sync and opens it", async () => {
        const user = userEvent.setup();
        const started: string[] = [];
        renderScreen(
            <DriftTable
                siteId="s1"
                baseUrl="https://shop.test"
                rows={[]}
                onStarted={(runId) => {
                    started.push(runId);
                }}
            />,
        );

        await user.click(screen.getByRole("button", { name: said.resync }));

        expect(held.sent).toStrictEqual([{ siteId: "s1" }]);
        expect(started).toEqual(["r-new"]);
    });

    it("says the stage of the sync under way and offers to open it", () => {
        held.active = aProgress({ runId: "r-live", stage: "store" });
        renderScreen(<DriftTable siteId="s1" baseUrl="https://shop.test" rows={[]} onStarted={() => undefined} />);

        expect(screen.getByRole("button", { name: copy.sync.syncing("reading the store") })).toHaveProperty(
            "disabled",
            true,
        );
        expect(screen.getByRole("button", { name: copy.sync.open })).toBeDefined();
    });
});
