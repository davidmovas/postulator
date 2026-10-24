import { fireEvent, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { renderScreen } from "../../testing/render.js";

const sent: unknown[] = [];
const started: string[] = [];

vi.mock("../../data/hooks/sync.js", () => ({
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

function asked() {
    renderScreen(
        <SyncDialog
            request={{ siteId: "s1", full: true, renderRescan: true }}
            onOpenChange={() => undefined}
            onStarted={(runId) => {
                started.push(runId);
            }}
        />,
    );
    return within(screen.getByRole("dialog", { name: said.againTitle }));
}

beforeEach(() => {
    sent.length = 0;
    started.length = 0;
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
});
