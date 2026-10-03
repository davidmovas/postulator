import { describe, expect, it } from "vitest";

import {
    cannibalizationReasons,
    importActions,
    importColumnUses,
    importFields,
    importFindingCodes,
    importRowTypes,
} from "../../generated/vocab.js";
import {
    actionLabel,
    actionTone,
    blocking,
    columnUseLabel,
    fieldChoices,
    fieldLabel,
    findingLabel,
    noField,
    productNote,
    reasonLabel,
    rowTypeLabel,
} from "./labels.js";

const vocabularies: readonly [string, readonly string[], (value: string) => string][] = [
    ["import field", importFields, fieldLabel],
    ["column use", importColumnUses.filter((use) => use !== "field"), (use) => columnUseLabel({ header: "", use })],
    ["finding code", importFindingCodes, findingLabel],
    ["import action", importActions, actionLabel],
    ["cannibalisation reason", cannibalizationReasons, reasonLabel],
    ["row type", importRowTypes, rowTypeLabel],
];

describe("what a product row of the preview says", () => {
    it.each([
        { name: "a page", page: { wpType: "page" }, want: null },
        { name: "a row the store holds no product for", page: { wpType: "product" }, want: "A product the store does not hold yet" },
        {
            name: "a product found by its slug",
            page: { wpType: "product", storeName: "Mak Liquid", matchedBy: "slug", plannedPath: "/mak/mak-liquid/" },
            want: "The store's Mak Liquid, found by its slug, from /mak/mak-liquid/",
        },
        {
            name: "a product found at its own address",
            page: { wpType: "product", storeName: "Mak Liquid", matchedBy: "path" },
            want: "The store's Mak Liquid, found by its address",
        },
    ])("$name", ({ page, want }) => {
        expect(productNote(page)).toBe(want);
    });
});

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

describe("what a column became", () => {
    it.each([
        [{ header: "URL", use: "field", field: "path" }, "Page path"],
        [{ header: "Category", use: "level" }, "Group"],
        [{ header: "Notes", use: "note" }, "Note for the writer"],
        [{ header: "Entity?", use: "ignored" }, "Ignored"],
    ])("words %j as %s", (column, want) => {
        expect(columnUseLabel(column)).toBe(want);
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
