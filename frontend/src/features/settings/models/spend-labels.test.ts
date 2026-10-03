import { describe, expect, test } from "vitest";

import { copy } from "../../../copy/index.js";
import { codes } from "../../../data/errors.js";
import { spendPurposes } from "../../../generated/vocab.js";
import {
    callStep,
    daysOf,
    defaultSpendDays,
    failureText,
    purposeLabel,
    spendDays,
    tierLabel,
} from "./spend-labels.js";

const said = copy.settings.models.spend;

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

describe("why a call failed, in a sentence", () => {
    test("an exhausted account says to add credit", () => {
        expect(failureText("NEEDS_HUMAN")).toMatch(/out of credit/);
        expect(failureText("NEEDS_HUMAN")).toMatch(/billing/);
    });

    test("a refused key says where to set it again", () => {
        expect(failureText("UNAUTHORIZED")).toMatch(/Provider keys/);
    });

    test("a pause the provider asked for says it is tried again", () => {
        expect(failureText("RATE_LIMITED")).toMatch(/tries again/);
    });

    test("every code the application reports has its own sentence", () => {
        const sentences = codes.map(failureText);
        expect(new Set(sentences).size).toBe(codes.length);
        expect(sentences).not.toContain(said.failedUnknown);
    });

    test("a code nobody knows still reads as a sentence, never as the code", () => {
        expect(failureText("TEAPOT")).toBe(said.failedUnknown);
        expect(failureText("")).toBe(said.failedUnknown);
    });
});

describe("the step a call served, in words", () => {
    const cases: readonly { name: string; purpose: string; step: string; want: string }[] = [
        { name: "a run's writer", purpose: "run", step: "generate_body", want: copy.runs.steps.generate_body },
        { name: "a run kind's own step", purpose: "run", step: "relink_page", want: copy.runs.perKindSteps.relink_page },
        { name: "a run call that names no step", purpose: "run", step: "", want: "" },
        { name: "entities proposed from pages", purpose: "graph", step: "propose_from_pages", want: said.recent.proposals.propose_from_pages },
        { name: "entities proposed from keywords", purpose: "graph", step: "propose_from_keywords", want: said.recent.proposals.propose_from_keywords },
        { name: "related entities proposed", purpose: "graph", step: "propose_related", want: said.recent.proposals.propose_related },
        { name: "an assistant round says no more than its purpose", purpose: "chat", step: "chat", want: "" },
        { name: "a title says no more than its purpose", purpose: "title", step: "title", want: "" },
        { name: "a provider test says no more than its purpose", purpose: "probe", step: "test_provider", want: "" },
        { name: "a page audit says no more than its purpose", purpose: "audit", step: "judge", want: "" },
        { name: "anything else keeps what the ledger recorded", purpose: "other", step: "generate_images", want: copy.runs.steps.generate_images },
    ];

    for (const tc of cases) {
        test(tc.name, () => {
            expect(callStep(tc.purpose, tc.step)).toBe(tc.want);
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
