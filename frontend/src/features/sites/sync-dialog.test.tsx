import { fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { SyncProgress } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { aProgress } from "../../testing/sync.js";

const sent: unknown[] = [];
const started: string[] = [];
const held: { active: SyncProgress | null } = { active: null };

vi.mock("../../data/hooks/sync.js", () => ({
    useActiveSync: () => held.active,
    useSyncSite: () => ({
        mutate: (request: unknown, options?: { onSuccess?: (answer: { runId: string }) => void }) => {
            sent.push(request);
            options?.onSuccess?.({ runId: "r9" });
        },
        isPending: false,
    }),
}));

const { SyncDialog } = await import("./sync-dialog.js");

const said = copy.sites.sync;

function asked(title: string = said.againTitle) {
    renderScreen(
        <SyncDialog
            request={{ siteId: "s1", full: true, renderRescan: true }}
            onOpenChange={() => undefined}
            onStarted={(runId) => {
                started.push(runId);
            }}
        />,
    );
    return within(screen.getByRole("dialog", { name: title }));
}

beforeEach(() => {
    sent.length = 0;
    started.length = 0;
    held.active = null;
});

describe("the sync a rescan answer opens", () => {
    it("ticks both options the answer asked for and starts that sync", () => {
        const dialog = asked();

        expect((dialog.getByLabelText(said.full) as HTMLInputElement).checked).toBe(true);
        expect((dialog.getByLabelText(said.rescan) as HTMLInputElement).checked).toBe(true);
        fireEvent.click(dialog.getByRole("button", { name: said.start }));

        expect(sent).toStrictEqual([{ siteId: "s1", full: true, renderRescan: true }]);
        expect(started).toStrictEqual(["r9"]);
    });

    it("lets a person narrow the walk before it starts", () => {
        const dialog = asked();

        fireEvent.click(dialog.getByLabelText(said.full));
        fireEvent.click(dialog.getByRole("button", { name: said.start }));

        expect(sent).toStrictEqual([{ siteId: "s1", full: false, renderRescan: true }]);
    });

    it("opens the sync under way instead of asking for a second one", async () => {
        const user = userEvent.setup();
        held.active = aProgress({ runId: "r-live", stage: "derive", pulled: 400 });
        const dialog = asked(copy.sync.underWay.title);

        expect(dialog.getByText("Deriving the page tree · 400 pages read")).toBeDefined();
        expect(dialog.queryByLabelText(said.full)).toBeNull();
        await user.click(dialog.getByRole("button", { name: copy.sync.open }));

        expect(sent).toEqual([]);
        expect(started).toEqual(["r-live"]);
    });
});
