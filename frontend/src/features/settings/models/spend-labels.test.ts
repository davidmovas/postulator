import { describe, expect, test } from "vitest";

import { spendPurposes } from "../../../generated/vocab.js";
import { daysOf, defaultSpendDays, purposeLabel, spendDays, tierLabel } from "./spend-labels.js";

describe("what the money was spent on, in words", () => {
    const cases: readonly { purpose: string; want: string }[] = [
        { purpose: "run", want: "Page runs" },
        { purpose: "chat", want: "Assistant" },
        { purpose: "title", want: "Titles" },
        { purpose: "probe", want: "Provider test" },
        { purpose: "graph", want: "Entity proposals" },
        { purpose: "audit", want: "Page audits" },
        { purpose: "other", want: "Other" },
        { purpose: "batch", want: "Other" },
        { purpose: "", want: "Other" },
    ];

    for (const tc of cases) {
        test(`${tc.purpose === "" ? "no purpose" : tc.purpose} reads ${tc.want}`, () => {
            expect(purposeLabel(tc.purpose)).toBe(tc.want);
        });
    }

    test("every purpose the ledger knows has its own words", () => {
        const labels = spendPurposes.map(purposeLabel);
        expect(new Set(labels).size).toBe(spendPurposes.length);
    });
});

describe("the tier a call was served on, in words", () => {
    const cases: readonly { tier: string; want: string }[] = [
        { tier: "default", want: "Standard" },
        { tier: "flex", want: "Flex" },
        { tier: "priority", want: "priority" },
    ];

    for (const tc of cases) {
        test(tc.tier, () => {
            expect(tierLabel(tc.tier)).toBe(tc.want);
        });
    }
});

describe("the period the spend covers", () => {
    test("offers a week, a month and a quarter, a month first", () => {
        expect(spendDays).toEqual([7, 30, 90]);
        expect(defaultSpendDays).toBe(30);
    });

    const cases: readonly { picked: string; want: number }[] = [
        { picked: "7", want: 7 },
        { picked: "30", want: 30 },
        { picked: "90", want: 90 },
        { picked: "12", want: 30 },
        { picked: "", want: 30 },
    ];

    for (const tc of cases) {
        test(`"${tc.picked}" reads ${tc.want} days`, () => {
            expect(daysOf(tc.picked)).toBe(tc.want);
        });
    }
});
