import { screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { RunReport, SpendReport, SpendSlice } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";

const said = copy.reports.runs.costByStep;

const held: { spend: SpendReport | undefined; askedFor: (string | null)[] } = { spend: undefined, askedFor: [] };

const report: RunReport = {
    runId: "r1",
    siteId: "s1",
    kind: "generate",
    status: "completed",
    stats: { items: 2, done: 2, failed: 0, tokens: 48_000, usd: 1 },
    items: [],
};

vi.mock("../../data/hooks/reports.js", () => ({
    useRunReport: () => ({ data: report, isPending: false, error: null }),
}));

vi.mock("../../data/hooks/pages.js", () => ({
    usePageTree: () => ({ data: { roots: [] }, isPending: false, error: null }),
}));

vi.mock("../../data/hooks/models.js", () => ({
    useRunSpend: (runId: string | null) => {
        held.askedFor.push(runId);
        return { data: held.spend, isPending: held.spend === undefined, error: null };
    },
}));

const { RunReportPanel } = await import("./run-panel.js");

function slice(overrides: Partial<SpendSlice>): SpendSlice {
    return {
        purpose: "run",
        provider: "openai",
        model: "gpt-5.6-terra",
        tier: "default",
        step: "",
        calls: 1,
        failed: 0,
        input: 1000,
        cachedInput: 0,
        cacheWrite: 0,
        output: 0,
        reasoning: 0,
        usd: 0,
        ...overrides,
    };
}

function spendOf(slices: SpendSlice[]): SpendReport {
    const usd = slices.reduce((sum, held) => sum + held.usd, 0);
    return {
        since: null,
        days: 0,
        runId: "r1",
        totals: {
            calls: slices.reduce((sum, held) => sum + held.calls, 0),
            failed: slices.reduce((sum, held) => sum + held.failed, 0),
            input: 0,
            cachedInput: 0,
            cacheWrite: 0,
            output: 0,
            reasoning: 0,
            usd,
            cachedShare: 0,
            reasoningShare: 0,
            flexShare: 0,
        },
        slices,
    };
}

function steps(): HTMLElement[] {
    return within(screen.getByRole("list", { name: said.title })).getAllByRole("listitem");
}

function show(): void {
    renderScreen(<RunReportPanel siteId="s1" runId="r1" />);
}

beforeEach(() => {
    held.askedFor = [];
    held.spend = spendOf([
        slice({ step: "generate_body", tier: "flex", calls: 3, output: 6000, reasoning: 3000, usd: 0.6 }),
        slice({ step: "generate_meta", model: "gpt-5.6-luna", calls: 1, output: 300, usd: 0.1 }),
        slice({ step: "generate_body", calls: 1, failed: 1, output: 2000, reasoning: 1000, usd: 0.2 }),
        slice({ step: "judge", calls: 2, output: 500, reasoning: 100, usd: 0.1 }),
    ]);
});

describe("the run report's cost by step", () => {
    it("asks for the spend of the run it shows", () => {
        show();

        expect(held.askedFor.at(-1)).toBe("r1");
    });

    it("prices each step of the run, dearest first", () => {
        show();

        expect(steps().map((step) => within(step).getByRole("heading").textContent)).toEqual([
            copy.runs.steps.generate_body,
            copy.runs.steps.judge,
            copy.runs.steps.generate_meta,
        ]);
        const [writer] = steps();
        if (writer === undefined) {
            throw new Error("a step expected");
        }
        expect(within(writer).getByText("$0.80")).toBeDefined();
        expect(within(writer).getByText(said.ofRun("80%"))).toBeDefined();
    });

    it("counts each step's calls and says how much of what it wrote was reasoning", () => {
        show();
        const [writer, judge, meta] = steps();
        if (writer === undefined || judge === undefined || meta === undefined) {
            throw new Error("three steps expected");
        }

        expect(within(writer).getByText(said.calls(4))).toBeDefined();
        expect(within(writer).getByText(said.reasoning("50%"))).toBeDefined();
        expect(within(writer).getByText(said.failed(1))).toBeDefined();
        expect(within(judge).getByText(said.reasoning("20%"))).toBeDefined();
        expect(within(meta).getByText(said.calls(1))).toBeDefined();
        expect(within(meta).getByText(said.reasoning("0%"))).toBeDefined();
        expect(within(meta).queryByText(said.failed(0))).toBeNull();
    });

    it("names the calls a run recorded without a step", () => {
        held.spend = spendOf([slice({ step: "", usd: 0.05 })]);
        show();

        expect(within(steps()[0] ?? document.body).getByRole("heading").textContent).toBe(said.unnamed);
    });

    it("says when no step of the run called a model", () => {
        held.spend = spendOf([]);
        show();

        expect(screen.getByText(said.none)).toBeDefined();
        expect(screen.queryByRole("list", { name: said.title })).toBeNull();
    });

    it("shows a loading state before the spend arrives", () => {
        held.spend = undefined;
        show();

        expect(screen.getByRole("status", { name: said.loading })).toBeDefined();
    });
});
