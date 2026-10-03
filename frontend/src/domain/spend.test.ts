import { describe, expect, test } from "vitest";

import type { SpendSlice } from "../data/types.js";
import { count, modelRows, percent, purposeRows, share, stepRows } from "./spend.js";

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
        output: 200,
        reasoning: 0,
        usd: 0.01,
        ...overrides,
    };
}

describe("a share", () => {
    const cases: readonly { name: string; part: number; whole: number; want: number }[] = [
        { name: "nothing of nothing", part: 0, whole: 0, want: 0 },
        { name: "something of nothing", part: 5, whole: 0, want: 0 },
        { name: "a quarter", part: 25, whole: 100, want: 0.25 },
        { name: "more than the whole is the whole", part: 140, whole: 100, want: 1 },
        { name: "a negative part is none", part: -3, whole: 100, want: 0 },
    ];

    for (const tc of cases) {
        test(tc.name, () => {
            expect(share(tc.part, tc.whole)).toBe(tc.want);
        });
    }
});

describe("a percentage the owner reads", () => {
    const cases: readonly { name: string; fraction: number; want: string }[] = [
        { name: "none", fraction: 0, want: "0%" },
        { name: "not a number", fraction: Number.NaN, want: "0%" },
        { name: "a sliver is never shown as none", fraction: 0.004, want: "<1%" },
        { name: "one percent", fraction: 0.01, want: "1%" },
        { name: "rounded to a whole percent", fraction: 0.8649, want: "86%" },
        { name: "nearly all is never shown as all", fraction: 0.996, want: ">99%" },
        { name: "all", fraction: 1, want: "100%" },
        { name: "more than all is all", fraction: 1.3, want: "100%" },
    ];

    for (const tc of cases) {
        test(tc.name, () => {
            expect(percent(tc.fraction)).toBe(tc.want);
        });
    }
});

describe("a count of calls", () => {
    const cases: readonly { name: string; value: number; want: string }[] = [
        { name: "none", value: 0, want: "0" },
        { name: "a few", value: 24, want: "24" },
        { name: "thousands are grouped", value: 1204, want: "1,204" },
        { name: "never shortened", value: 1_250_000, want: "1,250,000" },
    ];

    for (const tc of cases) {
        test(tc.name, () => {
            expect(count(tc.value)).toBe(tc.want);
        });
    }
});

describe("spend by purpose", () => {
    test("adds every model and tier of a purpose into one row", () => {
        const rows = purposeRows([
            slice({ purpose: "run", model: "gpt-5.6-terra", tier: "flex", calls: 10, failed: 1, usd: 2 }),
            slice({ purpose: "chat", model: "gpt-5.6-terra", calls: 30, usd: 1 }),
            slice({ purpose: "run", model: "gpt-5.6-luna", calls: 40, failed: 2, usd: 1 }),
        ]);

        expect(rows.map((row) => [row.purpose, row.calls, row.failed, row.usd, row.share])).toEqual([
            ["run", 50, 3, 3, 0.75],
            ["chat", 30, 0, 1, 0.25],
        ]);
    });

    test("puts the dearest purpose first", () => {
        const rows = purposeRows([
            slice({ purpose: "title", usd: 0.001 }),
            slice({ purpose: "graph", usd: 0.4 }),
            slice({ purpose: "chat", usd: 1.2 }),
        ]);

        expect(rows.map((row) => row.purpose)).toEqual(["chat", "graph", "title"]);
    });

    test("breaks a tie in spend by the number of calls, then by name", () => {
        const rows = purposeRows([
            slice({ purpose: "probe", usd: 0, calls: 1 }),
            slice({ purpose: "audit", usd: 0, calls: 1 }),
            slice({ purpose: "title", usd: 0, calls: 4 }),
        ]);

        expect(rows.map((row) => row.purpose)).toEqual(["title", "audit", "probe"]);
    });

    test("gives no share when nothing was spent", () => {
        const rows = purposeRows([slice({ purpose: "probe", usd: 0, failed: 1, calls: 0 })]);

        expect(rows[0]?.share).toBe(0);
    });

    test("reads no slices as no rows", () => {
        expect(purposeRows(null)).toEqual([]);
        expect(purposeRows([])).toEqual([]);
    });
});

describe("spend by model", () => {
    test("keeps a model's tiers apart and adds its purposes together", () => {
        const rows = modelRows([
            slice({ purpose: "run", model: "gpt-5.6-terra", tier: "flex", input: 9000, cachedInput: 6000, output: 3000, reasoning: 1200, usd: 0.5 }),
            slice({ purpose: "chat", model: "gpt-5.6-terra", tier: "default", input: 21000, cachedInput: 18000, output: 400, usd: 0.3 }),
            slice({ purpose: "audit", model: "gpt-5.6-terra", tier: "flex", input: 1000, cachedInput: 0, output: 100, reasoning: 50, usd: 0.1 }),
            slice({ purpose: "run", model: "gpt-5.6-luna", tier: "default", calls: 3, usd: 0.05 }),
        ]);

        expect(
            rows.map((row) => [row.model, row.tier, row.calls, row.input, row.cachedInput, row.output, row.reasoning]),
        ).toEqual([
            ["gpt-5.6-terra", "flex", 2, 10000, 6000, 3100, 1250],
            ["gpt-5.6-terra", "default", 1, 21000, 18000, 400, 0],
            ["gpt-5.6-luna", "default", 3, 1000, 0, 200, 0],
        ]);
        expect(rows[0]?.usd).toBeCloseTo(0.6);
    });

    test("tells two providers serving the same model name apart", () => {
        const rows = modelRows([
            slice({ provider: "openai", model: "shared", usd: 0.2 }),
            slice({ provider: "other", model: "shared", usd: 0.1 }),
        ]);

        expect(rows.map((row) => row.provider)).toEqual(["openai", "other"]);
    });
});

describe("a run's spend by step", () => {
    test("adds a step's models and tiers and says what it cost of the run", () => {
        const rows = stepRows([
            slice({ step: "generate_body", tier: "flex", calls: 3, output: 6000, reasoning: 3000, usd: 0.6 }),
            slice({ step: "generate_meta", model: "gpt-5.6-luna", calls: 1, output: 300, reasoning: 0, usd: 0.1 }),
            slice({ step: "generate_body", tier: "default", calls: 1, failed: 1, output: 2000, reasoning: 1000, usd: 0.2 }),
            slice({ step: "judge", calls: 2, output: 500, reasoning: 100, usd: 0.1 }),
        ]);

        expect(rows.map((row) => [row.step, row.calls, row.failed, row.output, row.reasoning])).toEqual([
            ["generate_body", 4, 1, 8000, 4000],
            ["judge", 2, 0, 500, 100],
            ["generate_meta", 1, 0, 300, 0],
        ]);
        expect(rows.map((row) => Math.round(row.share * 100))).toEqual([80, 10, 10]);
        expect(rows.map((row) => row.reasoningShare)).toEqual([0.5, 0.2, 0]);
    });

    test("names a call recorded without a step as an empty step", () => {
        const rows = stepRows([slice({ step: "", usd: 0.3 })]);

        expect(rows.map((row) => row.step)).toEqual([""]);
    });
});
