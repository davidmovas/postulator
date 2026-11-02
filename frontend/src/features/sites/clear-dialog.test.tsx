import { fireEvent, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { SiteDataHeld } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { aSite } from "../../testing/sites.js";

const sent: unknown[] = [];
const closed: string[] = [];
const counted: { held: SiteDataHeld; busy: string[] } = {
    held: { pages: 0, findings: 0, entities: 0, edges: 0, runs: 0, unfinished: 0, chats: 0 },
    busy: [],
};
const refused: { error: unknown } = { error: null };

vi.mock("../../data/hooks/sites.js", () => ({
    useSiteDataHeld: () => ({ data: { held: counted.held, busy: counted.busy }, isPending: false }),
    useClearSiteData: () => ({
        mutate: (request: unknown, options?: { onSuccess?: () => void }) => {
            sent.push(request);
            options?.onSuccess?.();
        },
        isPending: false,
        error: refused.error,
    }),
}));

const { ClearDataDialog, chosen, nothingChosen } = await import("./clear-dialog.js");

const said = copy.sites.clear;

function opened() {
    renderScreen(
        <ClearDataDialog
            site={aSite({ id: "s1", name: "Shop" })}
            onClose={() => {
                closed.push("closed");
            }}
        />,
    );
    return within(screen.getByRole("dialog", { name: said.title }));
}

function box(dialog: ReturnType<typeof opened>, label: string): HTMLInputElement {
    return dialog.getByLabelText(label) as HTMLInputElement;
}

beforeEach(() => {
    sent.length = 0;
    closed.length = 0;
    counted.held = { pages: 2140, findings: 312, entities: 1906, edges: 2410, runs: 48, unfinished: 2, chats: 6 };
    counted.busy = [];
    refused.error = null;
});

describe("what a clear takes with it", () => {
    it.each([
        { name: "nothing", asked: { model: false, graph: false, history: false }, empty: true, history: false },
        { name: "the model", asked: { model: true, graph: false, history: false }, empty: false, history: true },
        { name: "the graph", asked: { model: false, graph: true, history: false }, empty: false, history: false },
        { name: "the history", asked: { model: false, graph: false, history: true }, empty: false, history: true },
    ])("$name", ({ asked, empty, history }) => {
        expect(nothingChosen(chosen(asked))).toBe(empty);
        expect(chosen(asked).history).toBe(history);
    });
});

describe("the clear data dialog", () => {
    it("says what each group holds before anything is chosen", () => {
        const dialog = opened();

        expect(dialog.getByText(said.modelHint(2140, 312))).toBeDefined();
        expect(dialog.getByText(said.graphHint(1906, 2410))).toBeDefined();
        expect(dialog.getByText(said.historyHint(48, 6), { exact: false })).toBeDefined();
        expect(dialog.getByText(said.unfinished(2), { exact: false })).toBeDefined();
        expect(dialog.getByRole("button", { name: said.confirm }).hasAttribute("disabled")).toBe(true);
    });

    it("ticks and locks the runs and chats when the model is chosen, and sends all three", () => {
        const dialog = opened();

        fireEvent.click(box(dialog, said.model));

        expect(box(dialog, said.history).checked).toBe(true);
        expect(box(dialog, said.history).disabled).toBe(true);
        expect(dialog.getByText(said.historyTaken)).toBeDefined();
        fireEvent.click(dialog.getByRole("button", { name: said.confirm }));

        expect(sent).toStrictEqual([{ id: "s1", model: true, graph: false, history: true }]);
        expect(closed).toStrictEqual(["closed"]);
    });

    it("clears the graph alone", () => {
        const dialog = opened();

        fireEvent.click(box(dialog, said.graph));
        fireEvent.click(dialog.getByRole("button", { name: said.confirm }));

        expect(sent).toStrictEqual([{ id: "s1", model: false, graph: true, history: false }]);
    });

    it("refuses while a run of the site is under way and points at Runs", () => {
        counted.busy = ["r1"];
        const dialog = opened();

        fireEvent.click(box(dialog, said.graph));

        expect(dialog.getByText(copy.sites.busy)).toBeDefined();
        expect(dialog.getByRole("link", { name: copy.sites.openRuns }).getAttribute("href")).toBe("/s/s1/runs");
        fireEvent.click(dialog.getByRole("button", { name: said.confirm }));
        expect(sent).toStrictEqual([]);
    });

    it("shows the busy refusal the server gave instead of the generic conflict", () => {
        refused.error = new Error("refused", {
            cause: { code: "CONFLICT", message: "a run of this site is under way", details: { code: "site_busy" } },
        });
        const dialog = opened();

        expect(dialog.getByText(copy.sites.busy)).toBeDefined();
        expect(dialog.queryByText(copy.errors.CONFLICT)).toBeNull();
    });
});
