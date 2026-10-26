import { describe, expect, it } from "vitest";

import { aPlan, aStep } from "./fixture.js";
import {
    choicesOf,
    costOf,
    dependentsOf,
    everyStep,
    planIdOf,
    proposedPlanOf,
    requirementsOf,
    ticked,
} from "./model.js";

const steps = [
    aStep(1),
    aStep(2, { dependsOn: [1] }),
    aStep(3, { dependsOn: [2] }),
    aStep(4),
    aStep(5, { dependsOn: [3, 4] }),
];

describe("a step's dependents and requirements", () => {
    it.each([
        ["s1", ["s2", "s3", "s5"]],
        ["s2", ["s3", "s5"]],
        ["s4", ["s5"]],
        ["s5", []],
        ["missing", []],
    ])("unticking %s reaches every step that waits on it, transitively", (stepId, reached) => {
        expect([...dependentsOf(steps, stepId)].sort()).toStrictEqual(reached);
    });

    it.each([
        ["s5", ["s1", "s2", "s3", "s4"]],
        ["s3", ["s1", "s2"]],
        ["s1", []],
    ])("ticking %s needs every step before it that it waits on", (stepId, needed) => {
        expect([...requirementsOf(steps, stepId)].sort()).toStrictEqual(needed);
    });
});

describe("ticking and unticking", () => {
    const all = everyStep(aPlan(steps));

    it("unticks the dependents of an unticked step and leaves the others", () => {
        expect([...ticked(steps, all, "s2", false)].sort()).toStrictEqual(["s1", "s4"]);
    });

    it("ticks what a ticked step needs", () => {
        const none = new Set<string>();
        expect([...ticked(steps, none, "s3", true)].sort()).toStrictEqual(["s1", "s2", "s3"]);
    });
});

describe("the choices sent to ConfirmPlan", () => {
    const edited = [
        aStep(1, {
            edits: [
                { name: "anchor", value: "kits" },
                { name: "goal", value: "Sell" },
            ],
        }),
        aStep(2, { edits: [{ name: "anchor", value: "mugs" }] }),
    ];

    it("sends only the values the person changed, and only for approved steps", () => {
        const drafts = new Map<string, Record<string, string>>([
            ["s1", { anchor: "peptide kits", goal: "Sell" }],
            ["s2", { anchor: "cups" }],
        ]);
        expect(choicesOf(edited, new Set(["s1"]), drafts)).toStrictEqual([
            { stepId: "s1", approve: true, edits: { anchor: "peptide kits" } },
            { stepId: "s2", approve: false },
        ]);
    });

    it("sends no edits when nothing changed", () => {
        expect(choicesOf(edited, new Set(["s1", "s2"]), new Map())).toStrictEqual([
            { stepId: "s1", approve: true },
            { stepId: "s2", approve: true },
        ]);
    });

    it("declines every step when nothing is ticked", () => {
        expect(choicesOf(edited, new Set(), new Map()).every((choice) => !choice.approve)).toBe(true);
    });
});

describe("the cost of the chosen steps", () => {
    it("adds the estimates of the ticked steps only", () => {
        const priced = [
            aStep(1, { estimateUsd: 0.25, estimateTokens: 1000 }),
            aStep(2, { estimateUsd: 0.5, estimateTokens: 3000 }),
            aStep(3),
        ];
        expect(costOf(priced, new Set(["s1", "s3"]))).toStrictEqual({ usd: 0.25, tokens: 1000, priced: 1 });
    });
});

describe("a plans tool result", () => {
    const plan = aPlan([aStep(1)]);

    it.each([
        ["a whole answer", { status: "planProposed", planId: "p1", plan }, "p1"],
        [
            "a shortened answer",
            { truncated: true, totalBytes: 90000, result: { status: "planProposed", planId: "p1" } },
            "p1",
        ],
        ["a list answer", { plans: [plan] }, null],
        ["no answer", null, null],
    ])("finds the plan's id in %s", (_name, result, planId) => {
        expect(planIdOf(result)).toBe(planId);
    });

    it("takes the plan a whole answer carries, and never a shortened one", () => {
        expect(proposedPlanOf({ status: "planProposed", planId: "p1", plan })?.id).toBe("p1");
        expect(proposedPlanOf({ truncated: true, result: { status: "planProposed", planId: "p1", plan } })).toBeNull();
    });
});
