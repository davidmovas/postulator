import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { SpendReport, SpendSlice } from "../../../data/types.js";

const said = copy.settings.models.spend;

const held: { report: SpendReport | undefined; placeholder: boolean; asked: number[] } = {
    report: undefined,
    placeholder: false,
    asked: [],
};

vi.mock("../../../data/hooks/models.js", () => ({
    useSpendReport: (days: number) => {
        held.asked.push(days);
        return {
            data: held.report,
            isPending: held.report === undefined,
            isPlaceholderData: held.placeholder,
            error: null,
            refetch: () => Promise.resolve(),
        };
    },
}));

const { SpendPanel } = await import("./spend.js");

function slice(overrides: Partial<SpendSlice>): SpendSlice {
    return {
        purpose: "run",
        provider: "openai",
        model: "gpt-5.6-terra",
        tier: "default",
        step: "",
        calls: 0,
        failed: 0,
        input: 0,
        cachedInput: 0,
        cacheWrite: 0,
        output: 0,
        reasoning: 0,
        usd: 0,
        ...overrides,
    };
}

const month: SpendReport = {
    since: "2026-09-03T20:00:00Z",
    days: 30,
    runId: "",
    totals: {
        calls: 1204,
        failed: 3,
        input: 2_400_000,
        cachedInput: 2_064_000,
        cacheWrite: 0,
        output: 410_000,
        reasoning: 168_100,
        usd: 4.12,
        cachedShare: 0.86,
        reasoningShare: 0.41,
        flexShare: 0.62,
    },
    slices: [
        slice({
            tier: "flex",
            calls: 600,
            failed: 2,
            input: 1_500_000,
            cachedInput: 1_200_000,
            output: 300_000,
            reasoning: 150_000,
            usd: 2.55,
        }),
        slice({
            purpose: "chat",
            calls: 300,
            failed: 1,
            input: 800_000,
            cachedInput: 780_000,
            output: 60_000,
            usd: 1.2,
        }),
        slice({
            model: "gpt-5.6-luna",
            calls: 280,
            input: 90_000,
            cachedInput: 80_000,
            output: 45_000,
            reasoning: 18_100,
            usd: 0.35,
        }),
        slice({ purpose: "title", model: "gpt-5.6-luna", calls: 24, input: 10_000, cachedInput: 4_000, output: 5_000, usd: 0.02 }),
    ],
};

const nothing: SpendReport = {
    since: "2026-09-03T20:00:00Z",
    days: 30,
    runId: "",
    totals: {
        calls: 0,
        failed: 0,
        input: 0,
        cachedInput: 0,
        cacheWrite: 0,
        output: 0,
        reasoning: 0,
        usd: 0,
        cachedShare: 0,
        reasoningShare: 0,
        flexShare: 0,
    },
    slices: [],
};

function rowsOf(table: string): string[][] {
    const rows = within(screen.getByRole("table", { name: table })).getAllByRole("row").slice(1);
    return rows.map((row) => within(row).getAllByRole("cell").map((cell) => cell.textContent ?? ""));
}

beforeEach(() => {
    held.report = month;
    held.placeholder = false;
    held.asked = [];
});

describe("the spend panel", () => {
    it("says what was spent, on how many calls, and how many failed", () => {
        render(<SpendPanel />);

        expect(screen.getByText("$4.12")).toBeDefined();
        expect(screen.getByText(said.over(30))).toBeDefined();
        expect(screen.getByText(said.calls(1204))).toBeDefined();
        expect(screen.getByText(said.failed(3))).toBeDefined();
    });

    it("explains the cached, reasoning and flex shares in a line each", () => {
        render(<SpendPanel />);

        for (const [name, value] of [
            ["cached", "86%"],
            ["reasoning", "41%"],
            ["flex", "62%"],
        ] as const) {
            const tile = screen.getByRole("group", { name: said.shares[name].label });
            expect(within(tile).getByText(value)).toBeDefined();
            expect(within(tile).getByText(said.shares[name].note)).toBeDefined();
        }
    });

    it("lists what the money went on, dearest first, with its share", () => {
        render(<SpendPanel />);

        expect(rowsOf(said.byPurpose.title)).toEqual([
            ["Page runs", "880", "2", "$2.90", "70%"],
            ["Assistant", "300", "1", "$1.20", "29%"],
            ["Titles", "24", "0", "$0.02", "<1%"],
        ]);
    });

    it("lists each model on each tier with the tokens it read and wrote", () => {
        render(<SpendPanel />);

        expect(rowsOf(said.byModel.title)).toEqual([
            ["gpt-5.6-terra", "Flex", "600", "1.5M", "1.2M", "300K", "150K", "$2.55"],
            ["gpt-5.6-terra", "Standard", "300", "800K", "780K", "60K", "0", "$1.20"],
            ["gpt-5.6-luna", "Standard", "304", "100K", "84K", "50K", "18.1K", "$0.37"],
        ]);
    });

    it("asks for the period the owner picks", () => {
        render(<SpendPanel />);
        expect(held.asked.at(-1)).toBe(30);

        fireEvent.click(screen.getByRole("radio", { name: said.days(7) }));

        expect(held.asked.at(-1)).toBe(7);
        expect(screen.getByRole("radio", { name: said.days(7) }).getAttribute("aria-checked")).toBe("true");

        fireEvent.click(screen.getByRole("radio", { name: said.days(90) }));

        expect(held.asked.at(-1)).toBe(90);
    });

    it("keeps the last numbers, marked busy, while another period loads", () => {
        held.placeholder = true;
        render(<SpendPanel />);

        expect(screen.getByText("$4.12").closest("[aria-busy]")?.getAttribute("aria-busy")).toBe("true");
    });

    it("says nothing was spent instead of drawing empty tables", () => {
        held.report = nothing;
        render(<SpendPanel />);

        expect(screen.getByText(said.nothing(30))).toBeDefined();
        expect(screen.queryByRole("table")).toBeNull();
    });

    it("shows a loading state before the first answer", () => {
        held.report = undefined;
        render(<SpendPanel />);

        expect(screen.getByRole("status", { name: said.loading })).toBeDefined();
    });
});
