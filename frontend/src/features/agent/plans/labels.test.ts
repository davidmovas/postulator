import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { planStatuses, planStepStatuses } from "../../../generated/vocab.js";
import { editLabel, planStatusLabel, stepStatusLabel } from "./labels.js";

describe("the words of a plan", () => {
    it("has a word for every plan and step status", () => {
        for (const status of planStatuses) {
            expect(planStatusLabel(status)).not.toBe(status);
        }
        for (const status of planStepStatuses) {
            expect(stepStatusLabel(status)).not.toBe(status);
        }
    });

    it.each([
        ["goal", copy.plans.editNames["goal"]],
        ["type", copy.plans.editNames["type"]],
        ["maxWords", "Max words"],
        ["notes for /grind-size/", "Notes for /grind-size/"],
        ["anchor from /kits/ to /burr-grinders/", "Anchor from /kits/ to /burr-grinders/"],
    ])("names the edit %j %j, keeping a path as it is", (name, label) => {
        expect(editLabel(name)).toBe(label);
    });
});
