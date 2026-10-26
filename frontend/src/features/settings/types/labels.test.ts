import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import {
    anchorStrategies,
    fieldKeys,
    fieldProviders,
    pageTypePresets,
    pageTypeReasons,
    pageTypeRoles,
    pageTypeSources,
} from "../../../generated/vocab.js";
import {
    anchorLabel,
    decimal,
    fieldLabel,
    metaKeyOf,
    presetIcon,
    presetLabel,
    presetRoles,
    presetTone,
    providerLabel,
    reasonLabel,
    roleLabel,
    roleOf,
    sourceLabel,
} from "./labels.js";

const tables: readonly { name: string; values: readonly string[]; label: (value: string) => string }[] = [
    { name: "pageTypePresets", values: pageTypePresets, label: presetLabel },
    { name: "pageTypeRoles", values: pageTypeRoles, label: roleLabel },
    { name: "pageTypeSources", values: pageTypeSources, label: sourceLabel },
    { name: "pageTypeReasons", values: pageTypeReasons, label: reasonLabel },
    { name: "fieldKeys", values: fieldKeys, label: (value) => fieldLabel(value) },
    { name: "fieldProviders", values: fieldProviders, label: providerLabel },
    { name: "anchorStrategies", values: anchorStrategies, label: anchorLabel },
];

describe("the page type vocabulary in words", () => {
    for (const table of tables) {
        it(`names every one of ${table.name}`, () => {
            for (const value of table.values) {
                const label = table.label(value);
                expect(label, value).not.toBe("");
                expect(label, value).not.toContain("_");
            }
        });
    }

    it("shows a value this build does not know as it arrived, opened up", () => {
        expect(presetLabel("landing_page")).toBe("landing page");
        expect(reasonLabel("by_magic")).toBe("by magic");
        expect(fieldLabel("gallery_caption")).toBe("gallery caption");
    });
});

describe("presets as roles", () => {
    it("treats an untyped page and an unknown preset as untyped", () => {
        expect(roleOf(null)).toBe("untyped");
        expect(roleOf("")).toBe("untyped");
        expect(roleOf("pillar")).toBe("untyped");
        expect(roleOf("landing")).toBe("landing");
        expect(roleOf("hub")).toBe("hub");
        expect(presetLabel("")).toBe(copy.pageTypes.untyped);
    });

    it("draws every role with an icon of its own and a tone", () => {
        const icons = new Set(presetRoles.map((role) => presetIcon(role)));
        expect(icons.size).toBe(presetRoles.length);
        for (const role of presetRoles) {
            expect(presetTone(role)).not.toBe("");
        }
    });
});

describe("custom fields", () => {
    it("reads the meta key out of a custom field and names it by its label", () => {
        expect(metaKeyOf("meta:dosage")).toBe("dosage");
        expect(metaKeyOf("body")).toBeNull();
        expect(fieldLabel("meta:dosage", "Dosage")).toBe("Dosage");
        expect(fieldLabel("meta:dosage", " ")).toBe("dosage");
        expect(fieldLabel("seo_title")).toBe(copy.pageTypes.fields.seo_title);
    });

    it("shows a weight with its decimal point", () => {
        expect(decimal(1)).toBe("1.0");
        expect(decimal(0.55)).toBe("0.55");
    });
});
