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
    findingHeadline,
    pauseReasonShort,
    pauseReasonText,
    retryBlockedText,
    revertOutcomeLabel,
    runKindLabel,
    runStatusLabel,
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
        ["run kind", runKinds, runKindLabel],
        ["run status", runStatuses, runStatusLabel],
        ["item status", itemStatuses, runStatusLabel],
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
        expect(runKindLabel("teleport")).toBe("teleport");
        expect(runStatusLabel("melted")).toBe("melted");
        expect(artifactLabel("recipe_card")).toBe("recipe_card");
        expect(revertOutcomeLabel("vanished")).toBe("vanished");
    });
});

describe("the category findings of a run", () => {
    it.each([
        "page_categories_need_plugin",
        "categories_forbidden",
        "category_refused",
        "categories_not_taken",
        "revert_terms_kept",
        "revert_categories_kept",
        "categories_unread",
    ])("names %s in plain words", (code) => {
        spoken(findingHeadline(code) ?? "");
    });

    it("tells a person what to do when pages cannot carry categories", () => {
        expect(findingHeadline("page_categories_need_plugin")).toBe(
            "Categories on pages need the companion plugin 1.3.0 — update it and sync",
        );
    });

    it("leaves a finding it has no words for to its own message", () => {
        expect(findingHeadline("seo_meta_skipped")).toBeNull();
    });

    it.each(["trashed", "restored", "needs_human"])("says what a revert outcome %s means", (outcome) => {
        spoken(revertOutcomeLabel(outcome));
    });
});
