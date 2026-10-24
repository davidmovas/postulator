import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { imageSources, modelRoles, overrideScopes, stepNames, templateScopes } from "../../generated/vocab.js";
import {
    flagLabel,
    imageSourceLabel,
    overrideScopeLabel,
    percent,
    roleLabel,
    scopeLabel,
    scopeTone,
    stepLabel,
} from "./labels.js";

const tables: readonly { name: string; values: readonly string[]; label: (value: string) => string }[] = [
    { name: "stepNames", values: stepNames, label: stepLabel },
    { name: "modelRoles", values: modelRoles, label: roleLabel },
    { name: "templateScopes", values: templateScopes, label: scopeLabel },
    { name: "overrideScopes", values: overrideScopes, label: overrideScopeLabel },
    { name: "imageSources", values: imageSources, label: imageSourceLabel },
];

describe("labels", () => {
    for (const table of tables) {
        it(`names every ${table.name}`, () => {
            expect(table.values.length).toBeGreaterThan(0);
            for (const value of table.values) {
                const label = table.label(value);
                expect(label, value).not.toBe("");
                expect(label, value).not.toContain("_");
            }
        });
    }

    it("shows an unknown value as it arrived, with the underscores opened up", () => {
        expect(stepLabel("brand_new_step")).toBe("brand new step");
        expect(roleLabel("summariser")).toBe("summariser");
        expect(overrideScopeLabel("workspace")).toBe("workspace");
    });

    it("names an image source that was never chosen", () => {
        expect(imageSourceLabel("")).toBe(copy.templates.meta.noSource);
    });

    it("marks a site template and leaves a global one quiet", () => {
        expect(scopeTone("site")).toBe("accent");
        expect(scopeTone("global")).toBe("muted");
    });

    it("shows a share as a percentage", () => {
        expect(percent(0.025)).toBe("2.5%");
        expect(percent(0.03)).toBe("3%");
    });

    it("says whether a flag is on", () => {
        expect(flagLabel(true)).toBe(copy.templates.layer.on);
        expect(flagLabel(false)).toBe(copy.templates.layer.off);
    });
});
