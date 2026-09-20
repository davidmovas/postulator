import { copy } from "../../copy/index.js";
import type { JsonObject } from "../../domain/merge-patch.js";
import type { SpecPath } from "./patch.js";
import { paths, touches } from "./patch.js";

export type GroupKey = "sections" | "rules" | "models" | "recipe" | "overrides" | "policies";

export const groupKeys: readonly GroupKey[] = [
    "sections",
    "rules",
    "models",
    "recipe",
    "overrides",
    "policies",
];

const ruleRoots: readonly SpecPath[] = [
    paths.tone,
    ["length"],
    ["keywordRules"],
    ["linkRules"],
    ["metaRules"],
    ["images"],
];

const watched: Readonly<Record<GroupKey, readonly SpecPath[]>> = {
    sections: [paths.sections],
    rules: ruleRoots,
    models: [paths.profiles],
    recipe: [paths.recipe],
    overrides: [],
    policies: [],
};

export const groupTitles: Readonly<Record<GroupKey, string>> = {
    sections: copy.templates.sections.title,
    rules: copy.templates.editor.groups.rules,
    models: copy.templates.models.title,
    recipe: copy.templates.recipe.title,
    overrides: copy.templates.editor.groups.overrides,
    policies: copy.policies.title,
};

export function isGroup(candidate: string): candidate is GroupKey {
    return (groupKeys as readonly string[]).includes(candidate);
}

export function changedGroups(patch: JsonObject | null): ReadonlySet<GroupKey> {
    const changed = new Set<GroupKey>();
    if (patch === null || Object.keys(patch).length === 0) {
        return changed;
    }
    for (const key of groupKeys) {
        if (watched[key].some((path) => touches(patch, path))) {
            changed.add(key);
        }
    }
    if (changed.size > 0) {
        changed.add("overrides");
    }
    return changed;
}
