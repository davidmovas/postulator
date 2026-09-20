import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import {
    anchorStrategies,
    imageSources,
    modelRoles,
    overrideScopes,
    stepNames,
    templateScopes,
} from "../../generated/vocab.js";
import {
    anchorLabel,
    flagLabel,
    forbidsLabel,
    imageSourceLabel,
    overrideScopeLabel,
    pageKindLabel,
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
    { name: "anchorStrategies", values: anchorStrategies, label: anchorLabel },
    { name: "imageSources", values: imageSources, label: imageSourceLabel },
    { name: "pageKinds", values: Object.keys(copy.templates.pageKinds), label: pageKindLabel },
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
        expect(pageKindLabel("landing_page")).toBe("landing page");
        expect(anchorLabel("shortest")).toBe("shortest");
        expect(overrideScopeLabel("workspace")).toBe("workspace");
    });

    it("names an image source that was never chosen", () => {
        expect(imageSourceLabel("")).toBe(copy.templates.meta.noSource);
    });

    it("marks a site template and leaves a global one quiet", () => {
        expect(scopeTone("site")).toBe("accent");
        expect(scopeTone("global")).toBe("muted");
    });

    it("says what a policy forbids", () => {
        expect(forbidsLabel(true, true)).toBe(copy.policies.forbidsBoth);
        expect(forbidsLabel(true, false)).toBe(copy.policies.forbidsExternal);
        expect(forbidsLabel(false, true)).toBe(copy.policies.forbidsSelf);
        expect(forbidsLabel(false, false)).toBe(copy.policies.forbidsNone);
    });

    it("says whether a flag is on", () => {
        expect(flagLabel(true)).toBe(copy.templates.layer.on);
        expect(flagLabel(false)).toBe(copy.templates.layer.off);
    });
});
