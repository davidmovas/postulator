import { copy } from "../../copy/index.js";
import { flagLabel, roleLabel, stepLabel } from "./labels.js";
import { allRules } from "./rules-blocks.js";
import type { Rule } from "./rules-model.js";
import { shown } from "./rules-model.js";
import type { ProductDraft, ProfileDraft, SpecDraft, StepDraft } from "./spec.js";

export interface DiffRow {
    key: string;
    label: string;
    below: string;
    here: string;
    revert: Partial<SpecDraft>;
}

function revertOf(rule: Rule, below: SpecDraft): Partial<SpecDraft> {
    switch (rule.kind) {
        case "switch":
            return rule.write(rule.read(below));
        case "number":
            return rule.write(rule.read(below));
        case "text":
            return rule.write(rule.read(below));
        default:
            return rule.write(rule.read(below));
    }
}

function profileValue(profile: ProfileDraft | undefined): string {
    if (profile === undefined) {
        return copy.templates.layer.none;
    }
    return profile.model === "" ? profile.provider : `${profile.provider} · ${profile.model}`;
}

function paramText(step: StepDraft): string {
    const held = step.params;
    if (held === null) {
        return "";
    }
    const allow = held["allowErrors"];
    if (typeof allow === "boolean") {
        return ` · ${copy.templates.recipe.params.allowErrors} ${flagLabel(allow)}`;
    }
    const iterations = held["iterations"];
    if (typeof iterations === "number") {
        return ` · ${copy.templates.recipe.params.iterations} ${iterations}`;
    }
    return "";
}

function stepValue(step: StepDraft | undefined): string {
    if (step === undefined) {
        return copy.templates.layer.none;
    }
    return `${flagLabel(step.enabled)}${paramText(step)}`;
}

function sameStep(a: StepDraft | undefined, b: StepDraft | undefined): boolean {
    if (a === undefined || b === undefined) {
        return a === b;
    }
    return a.enabled === b.enabled && JSON.stringify(a.params) === JSON.stringify(b.params);
}

function sectionText(draft: SpecDraft): string {
    return copy.templates.sectionCount(draft.sections.length);
}

function productText(product: ProductDraft | null): string {
    if (product === null) {
        return copy.templates.product.none;
    }
    return copy.templates.product.summary(product.shortEnabled, product.specifications.length);
}

function productCopy(product: ProductDraft | null): ProductDraft | null {
    if (product === null) {
        return null;
    }
    return { ...product, specifications: product.specifications.map((row) => ({ ...row })) };
}

export function diffRows(below: SpecDraft, here: SpecDraft): readonly DiffRow[] {
    const rows: DiffRow[] = [];

    if (JSON.stringify(below.sections) !== JSON.stringify(here.sections)) {
        rows.push({
            key: "sections",
            label: copy.templates.overrides.sections,
            below: sectionText(below),
            here: sectionText(here),
            revert: { sections: below.sections.map((section) => ({ ...section, include: [...section.include] })) },
        });
    }

    if (JSON.stringify(below.product) !== JSON.stringify(here.product)) {
        rows.push({
            key: "product",
            label: copy.templates.product.title,
            below: productText(below.product),
            here: productText(here.product),
            revert: { product: productCopy(below.product) },
        });
    }

    for (const rule of allRules) {
        if (shown(rule, below) === shown(rule, here) && rule.read(below) === rule.read(here)) {
            continue;
        }
        rows.push({
            key: rule.key,
            label: rule.label,
            below: shown(rule, below),
            here: shown(rule, here),
            revert: revertOf(rule, below),
        });
    }

    const roles = [...new Set([...below.profiles, ...here.profiles].map((profile) => profile.role))];
    for (const role of roles) {
        const wasSet = below.profiles.find((profile) => profile.role === role);
        const isSet = here.profiles.find((profile) => profile.role === role);
        if (profileValue(wasSet) === profileValue(isSet)) {
            continue;
        }
        rows.push({
            key: `role-${role}`,
            label: roleLabel(role),
            below: profileValue(wasSet),
            here: profileValue(isSet),
            revert: {
                profiles:
                    wasSet === undefined
                        ? here.profiles.filter((profile) => profile.role !== role)
                        : here.profiles.some((profile) => profile.role === role)
                          ? here.profiles.map((profile) => (profile.role === role ? { ...wasSet } : profile))
                          : [...here.profiles, { ...wasSet }],
            },
        });
    }

    for (const step of here.recipe) {
        const was = below.recipe.find((held) => held.name === step.name);
        if (sameStep(was, step)) {
            continue;
        }
        rows.push({
            key: `step-${step.name}`,
            label: stepLabel(step.name),
            below: stepValue(was),
            here: stepValue(step),
            revert: {
                recipe: here.recipe.map((held) =>
                    held.name === step.name && was !== undefined ? { ...was } : held,
                ),
            },
        });
    }

    return rows;
}
