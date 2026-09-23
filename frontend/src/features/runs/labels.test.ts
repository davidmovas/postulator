import { describe, expect, it } from "vitest";

import {
    artifactKinds,
    itemStatuses,
    pauseReasons,
    perKindStepNames,
    retryBlockedReasons,
    runKinds,
    runStatuses,
    stepNames,
} from "../../generated/vocab.js";
import {
    artifactLabel,
    kindLabel,
    pauseReasonShort,
    pauseReasonText,
    retryBlockedText,
    statusLabel,
    stepLabel,
} from "./labels.js";

const spoken = (label: string): void => {
    expect(label).not.toBe("");
    expect(label).not.toMatch(/_/);
};

describe("the vocabulary never reaches the client as it is stored", () => {
    const cases: readonly (readonly [string, readonly string[], (value: string) => string])[] = [
        ["step", stepNames, stepLabel],
        ["step a kind owns", perKindStepNames, stepLabel],
        ["run kind", runKinds, kindLabel],
        ["run status", runStatuses, statusLabel],
        ["item status", itemStatuses, statusLabel],
        ["artifact kind", artifactKinds, artifactLabel],
        ["pause reason", pauseReasons, pauseReasonText],
        ["pause reason, short", pauseReasons, pauseReasonShort],
        ["retry block", retryBlockedReasons, retryBlockedText],
    ];

    for (const [name, values, label] of cases) {
        it(`speaks every ${name}`, () => {
            expect(values.length).toBeGreaterThan(0);
            for (const value of values) {
                spoken(label(value));
            }
        });
    }
});

describe("a value this build does not know", () => {
    it("is shown as it arrived rather than as a blank", () => {
        expect(stepLabel("summon_the_kraken")).toBe("summon_the_kraken");
        expect(kindLabel("teleport")).toBe("teleport");
        expect(statusLabel("melted")).toBe("melted");
        expect(artifactLabel("recipe_card")).toBe("recipe_card");
    });
});
