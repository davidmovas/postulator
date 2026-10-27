import { fireEvent, screen } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { renderScreen } from "../../testing/render.js";

const said = copy.runs.start;

interface Held {
    priced: Record<string, unknown>[];
    started: Record<string, unknown>[];
    error: unknown;
}

const held: Held = { priced: [], started: [], error: null };

const answered = { estimate: { tokens: 1200, usd: 0.4, findings: [] }, added: [], runId: "r1", fields: [] };

function mutation(into: Record<string, unknown>[]) {
    return {
        mutate: (request: Record<string, unknown>, options?: { onSuccess?: (answer: typeof answered) => void }) => {
            into.push(request);
            if (held.error === null) {
                options?.onSuccess?.(answered);
            }
        },
        reset: () => undefined,
        isPending: false,
        get error() {
            return held.error;
        },
    };
}

const paths: Readonly<Record<string, string>> = { "p-kits": "/kits/" };

const peptides = { pageId: "p-peptides", path: "/peptides/", title: "Peptides" };

vi.mock("../../data/hooks/runs.js", () => ({
    useEstimateRun: () => mutation(held.priced),
    useStartRun: () => mutation(held.started),
}));

vi.mock("../../data/hooks/templates.js", () => ({
    useTemplates: () => ({ data: undefined, isPending: false }),
    useResolvedTemplate: () => ({ data: undefined, isPending: false }),
}));

vi.mock("../../data/hooks/pages.js", () => ({
    usePagePaths: (ids: readonly string[]) =>
        new Map(ids.flatMap((id) => (paths[id] === undefined ? [] : [[id, paths[id]]]))),
    useSiteMap: (filter: { search?: string }) => ({
        data: filter.search === "" ? undefined : { pages: [{ nodes: { items: [{ page: peptides }] } }] },
        isPending: false,
        hasNextPage: false,
    }),
}));

vi.mock("./start-tree.js", () => ({
    TargetTree: () => null,
}));

const { StartRunDrawer } = await import("./start.js");

function drawer() {
    return renderScreen(
        <StartRunDrawer
            open={true}
            onOpenChange={() => undefined}
            siteId="s1"
            preselect={["p-kits"]}
            onStarted={() => undefined}
        />,
    );
}

function price(): Record<string, unknown> {
    fireEvent.click(screen.getByRole("button", { name: said.estimate }));
    return held.priced.at(-1) ?? {};
}

function openDirections(): void {
    fireEvent.click(screen.getByRole("button", { name: new RegExp("/kits/") }));
}

async function pickPeptides(): Promise<void> {
    fireEvent.click(screen.getByRole("button", { name: said.directives.addMustLink }));
    fireEvent.change(screen.getByRole("searchbox"), { target: { value: "pep" } });
    fireEvent.click(await screen.findByRole("option", { name: /Peptides/ }));
}

beforeAll(() => {
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
    Element.prototype.scrollIntoView = () => undefined;
});

beforeEach(() => {
    held.priced = [];
    held.started = [];
    held.error = null;
});

describe("the run's brief", () => {
    it("is folded away until it is asked for", () => {
        drawer();

        expect(screen.queryByText(said.brief.goal)).toBeNull();
        expect(screen.getByText(said.brief.filled(0, 6))).toBeDefined();

        fireEvent.click(screen.getByRole("button", { name: new RegExp(said.brief.title) }));
        expect(screen.getByLabelText(said.brief.goal)).toBeDefined();
        expect(screen.getByLabelText(said.brief.styleSample)).toBeDefined();
    });

    it("reaches the price with what was written", () => {
        drawer();

        fireEvent.click(screen.getByRole("button", { name: new RegExp(said.brief.title) }));
        fireEvent.change(screen.getByLabelText(said.brief.goal), { target: { value: "Sell the starter kit" } });

        expect(price()["brief"]).toStrictEqual({ goal: "Sell the starter kit" });
        expect(screen.getByText(said.brief.filled(1, 6))).toBeDefined();
    });

    it("refuses to price a brief over its limits", () => {
        drawer();

        fireEvent.click(screen.getByRole("button", { name: new RegExp(said.brief.title) }));
        fireEvent.change(screen.getByLabelText(said.brief.tone), { target: { value: "x".repeat(1001) } });

        expect(screen.getByText(said.brief.tooLong(1000))).toBeDefined();
        expect((screen.getByRole("button", { name: said.estimate }) as HTMLButtonElement).disabled).toBe(true);
    });
});

describe("a page's directions", () => {
    it("reach the price for that page", () => {
        drawer();

        openDirections();
        fireEvent.change(screen.getByLabelText(said.directives.notes), { target: { value: "Lead with the kit" } });

        expect(price()["directives"]).toStrictEqual([{ pageId: "p-kits", notes: "Lead with the kit" }]);
    });

    it("pick a page it must link to, with its anchor", async () => {
        drawer();

        openDirections();
        await pickPeptides();
        fireEvent.change(screen.getByRole("textbox", { name: said.directives.anchor }), {
            target: { value: "peptide basics" },
        });

        expect(price()["directives"]).toStrictEqual([
            { pageId: "p-kits", mustLink: [{ pageId: "p-peptides", anchor: "peptide basics" }] },
        ]);
    });
});

describe("a must-link the graph does not connect", () => {
    function refusedOnce(): void {
        held.error = Object.assign(new Error("refused"), {
            cause: {
                code: "INVALID",
                message: "/kits/ does not link to /peptides/: no edge of the entity graph connects their entities",
                details: { field: "directives.mustLink", from: "/kits/", to: "/peptides/" },
            },
        });
    }

    it("is said in plain words with the offer to add the edge, which prices again with it", async () => {
        drawer();
        openDirections();
        await pickPeptides();

        refusedOnce();
        price();

        expect(
            screen.getByText("/kits/ does not link to /peptides/: no edge of the entity graph connects their entities"),
        ).toBeDefined();
        held.error = null;
        fireEvent.click(screen.getByRole("checkbox", { name: said.mustLink.addEdges }));

        expect(held.priced.at(-1)?.["addEdges"]).toBe(true);
        expect(held.priced).toHaveLength(2);
    });
});
