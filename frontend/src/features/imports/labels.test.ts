import { describe, expect, it } from "vitest";

import {
    cannibalizationReasons,
    importActions,
    importFields,
    importFindingCodes,
} from "../../generated/vocab.js";
import {
    actionLabel,
    actionTone,
    blocking,
    fieldChoices,
    fieldLabel,
    findingLabel,
    noField,
    reasonLabel,
} from "./labels.js";

const vocabularies: readonly [string, readonly string[], (value: string) => string][] = [
    ["import field", importFields, fieldLabel],
    ["finding code", importFindingCodes, findingLabel],
    ["import action", importActions, actionLabel],
    ["cannibalisation reason", cannibalizationReasons, reasonLabel],
];

describe("every vocabulary reaches the screen as words", () => {
    it.each(vocabularies)("labels every %s", (_name, values, label) => {
        for (const value of values) {
            const written = label(value);
            expect(written).not.toBe("");
            expect(written).not.toContain("_");
        }
    });

    it.each(vocabularies)("shows an unknown %s as it arrived", (_name, _values, label) => {
        expect(label("something_new")).toBe("something_new");
    });
});

describe("the target field chooser", () => {
    it("offers every field plus the ignore choice", () => {
        expect(fieldChoices).toHaveLength(importFields.length + 1);
        expect(fieldChoices[0]?.value).toBe(noField);
        expect(fieldChoices.every((choice) => choice.label !== "")).toBe(true);
    });
});

describe("blocking findings", () => {
    it.each([
        ["bad_path", true],
        ["cycle", true],
        ["unknown_parent", true],
        ["cannibalization", false],
        ["duplicate_path", false],
        ["something_new", false],
    ])("reads %s as blocking %s", (code, want) => {
        expect(blocking(code)).toBe(want);
    });
});

describe("action tones", () => {
    it.each([
        ["create", "ok"],
        ["update", "info"],
        ["skip", "muted"],
    ] as const)("tints %s as %s", (action, want) => {
        expect(actionTone(action)).toBe(want);
    });
});
