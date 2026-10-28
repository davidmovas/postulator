import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { clearToasts, readToasts } from "../../data/toasts.js";
import type { SyncProgress } from "../../data/types.js";
import { aProgress } from "../../testing/sync.js";

type Answer = { runId: string } | { refused: unknown };

const held: { active: SyncProgress | null; answer: Answer; sent: unknown[] } = {
    active: null,
    answer: { runId: "r-new" },
    sent: [],
};

vi.mock("../../data/hooks/sync.js", () => ({
    useActiveSync: () => held.active,
    useSyncSite: () => ({
        mutate: (
            request: unknown,
            options?: { onSuccess?: (answered: { runId: string }) => void; onError?: (thrown: unknown) => void },
        ) => {
            held.sent.push(request);
            if ("runId" in held.answer) {
                options?.onSuccess?.(held.answer);
                return;
            }
            options?.onError?.(held.answer.refused);
        },
        isPending: false,
    }),
}));

const { SyncButton } = await import("./button.js");

const said = copy.sync;

function Where(): ReactElement {
    const location = useLocation();
    return <output aria-label="location">{location.pathname}</output>;
}

function shown(node: ReactElement): void {
    render(
        <MemoryRouter initialEntries={["/s/s1/overview"]}>
            {node}
            <Where />
        </MemoryRouter>,
    );
}

function location(): string {
    return screen.getByRole("status", { name: "location" }).textContent ?? "";
}

beforeEach(() => {
    held.active = null;
    held.answer = { runId: "r-new" };
    held.sent.length = 0;
    clearToasts();
});

describe("a sync button with no sync under way", () => {
    it("starts a sync of its site with the options it was given", async () => {
        const user = userEvent.setup();
        shown(<SyncButton siteId="s1" label="Sync site" options={{ full: true }} />);

        await user.click(screen.getByRole("button", { name: "Sync site" }));

        expect(held.sent).toStrictEqual([{ siteId: "s1", full: true }]);
    });

    it("hands the sync it started to the screen that asked for it", async () => {
        const user = userEvent.setup();
        const started: string[] = [];
        shown(
            <SyncButton
                siteId="s1"
                label="Sync site"
                onStarted={(runId) => {
                    started.push(runId);
                }}
            />,
        );

        await user.click(screen.getByRole("button", { name: "Sync site" }));

        expect(started).toEqual(["r-new"]);
    });

    it("says why a second sync was refused and opens the one under way", async () => {
        const user = userEvent.setup();
        const message =
            "a sync of Peptide shop is already under way; it reads every page, so a second one would only repeat it";
        held.answer = { refused: Object.assign(new Error(message), { cause: { code: "CONFLICT", message, details: { runId: "r-busy" } } }) };
        shown(<SyncButton siteId="s1" label="Sync site" />);

        await user.click(screen.getByRole("button", { name: "Sync site" }));

        expect(readToasts().map((toast) => toast.message)).toEqual([message]);
        expect(location()).toBe("/s/s1/runs/r-busy");
    });
});

describe("a sync button while a sync is under way", () => {
    it("says the stage the sync is in and cannot start another", () => {
        held.active = aProgress({ runId: "r-sync", stage: "render" });
        shown(<SyncButton siteId="s1" label="Sync site" />);

        const busy = screen.getByRole("button", { name: said.syncing("visiting the pages") });
        expect(busy).toHaveProperty("disabled", true);
        expect(screen.queryByRole("button", { name: "Sync site" })).toBeNull();
    });

    it("keeps a crowded header short by offering the sync under way as an icon", async () => {
        const user = userEvent.setup();
        held.active = aProgress({ runId: "r-sync", stage: "content" });
        shown(<SyncButton siteId="s1" label="Sync site" compact={true} />);

        const open = screen.getByRole("button", { name: said.open });
        expect(open.textContent).toBe("");
        await user.click(open);

        expect(location()).toBe("/s/s1/runs/r-sync");
    });

    it("opens the sync under way", async () => {
        const user = userEvent.setup();
        held.active = aProgress({ runId: "r-sync", stage: "content" });
        shown(<SyncButton siteId="s1" label="Sync site" />);

        await user.click(screen.getByRole("button", { name: said.open }));

        expect(location()).toBe("/s/s1/runs/r-sync");
        expect(held.sent).toEqual([]);
    });
});
