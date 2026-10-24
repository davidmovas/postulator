import type { CatalogField, FieldScope, LinkPolicy, LinkRules, PageType, SetScopeRequest } from "../../../data/types.js";
import { anchorStrategies, isOneOf, pageTypePresets, scopableFieldKeys } from "../../../generated/vocab.js";
import { metaKeyOf, metaPrefix } from "./labels.js";

export interface ScopeDraft {
    enabled: boolean;
    aiWrites: boolean;
    instruction: string;
    minLength: number;
    maxLength: number;
    rewrites: boolean;
    links: LinkPolicy;
    label: string;
    provider: string;
}

type Anchor = LinkPolicy["anchorStrategy"];

const productFields: ReadonlySet<string> = new Set([
    "product_description",
    "product_short_description",
    "variation_description",
    "upsells",
    "cross_sells",
]);

const notOnProducts: ReadonlySet<string> = new Set(["body", "excerpt"]);

export function draftOf(scope: FieldScope): ScopeDraft {
    return {
        enabled: scope.enabled,
        aiWrites: scope.aiWrites,
        instruction: scope.instruction,
        minLength: scope.minLength,
        maxLength: scope.maxLength,
        rewrites: scope.rewrites,
        links: { ...scope.links, rules: { ...scope.links.rules } },
        label: scope.label,
        provider: scope.provider,
    };
}

export function requestOf(pageTypeId: string, field: string, draft: ScopeDraft): SetScopeRequest {
    return {
        pageTypeId,
        field,
        enabled: draft.enabled,
        aiWrites: draft.aiWrites,
        instruction: draft.instruction.trim(),
        minLength: draft.minLength,
        maxLength: draft.maxLength,
        rewrites: draft.rewrites,
        links: draft.links,
        label: draft.label.trim(),
        provider: draft.provider,
    };
}

function sameRules(left: LinkRules, right: LinkRules): boolean {
    return (
        left.upDepth === right.upDepth &&
        left.downLinks === right.downLinks &&
        left.siblingMinWeight === right.siblingMinWeight &&
        left.maxLinks === right.maxLinks &&
        left.maxPerTarget === right.maxPerTarget &&
        left.parentLinkWithinParagraphs === right.parentLinkWithinParagraphs &&
        left.childrenSection === right.childrenSection
    );
}

function samePolicy(left: LinkPolicy, right: LinkPolicy): boolean {
    return (
        left.links === right.links &&
        left.forbidExternal === right.forbidExternal &&
        left.forbidSelf === right.forbidSelf &&
        left.anchorStrategy === right.anchorStrategy &&
        sameRules(left.rules, right.rules)
    );
}

export function sameDraft(left: ScopeDraft, right: ScopeDraft): boolean {
    return (
        left.enabled === right.enabled &&
        left.aiWrites === right.aiWrites &&
        left.instruction === right.instruction &&
        left.minLength === right.minLength &&
        left.maxLength === right.maxLength &&
        left.rewrites === right.rewrites &&
        left.label === right.label &&
        left.provider === right.provider &&
        samePolicy(left.links, right.links)
    );
}

export type LengthProblem = "negative" | "maxBelowMin";

export function lengthProblem(draft: ScopeDraft): LengthProblem | null {
    if (draft.minLength < 0 || draft.maxLength < 0) {
        return "negative";
    }
    return draft.maxLength > 0 && draft.maxLength < draft.minLength ? "maxBelowMin" : null;
}

function asAnchor(value: string): Anchor {
    return value as Anchor;
}

export function anchorOf(value: string, fallback: Anchor): Anchor {
    return isOneOf(anchorStrategies, value) ? asAnchor(value) : fallback;
}

export function aiWritableOf(catalog: readonly CatalogField[], field: string): boolean {
    const base = field.split(":")[0] ?? field;
    const held = catalog.find((entry) => entry.key === field) ?? catalog.find((entry) => entry.key === base);
    return held === undefined || held.aiWritable;
}

export function linkField(preset: string): string {
    return preset === "product" ? "product_description" : "body";
}

export function carriesLinks(field: string, preset: string): boolean {
    return field === linkField(preset);
}

export function usualFor(scope: FieldScope, preset: string): boolean {
    if (scope.enabled || scope.stored || metaKeyOf(scope.field) !== null) {
        return true;
    }
    if (preset === "product") {
        return !notOnProducts.has(scope.field);
    }
    return !productFields.has(scope.field);
}

function catalogRank(field: string): number {
    const at = scopableFieldKeys.indexOf(field as (typeof scopableFieldKeys)[number]);
    return at < 0 ? scopableFieldKeys.length : at;
}

export function orderedScopes(scopes: readonly FieldScope[]): FieldScope[] {
    return [...scopes].sort((left, right) => {
        const rank = catalogRank(left.field) - catalogRank(right.field);
        if (rank !== 0) {
            return rank;
        }
        return (left.label || left.field).localeCompare(right.label || right.field, undefined, { sensitivity: "base" });
    });
}

export function pickSite(asked: string | null, last: string | null, ids: readonly string[]): string | null {
    if (asked !== null && ids.includes(asked)) {
        return asked;
    }
    if (last !== null && ids.includes(last)) {
        return last;
    }
    return ids[0] ?? null;
}

function presetRank(preset: string): number {
    const at = (pageTypePresets as readonly string[]).indexOf(preset);
    return at < 0 ? pageTypePresets.length : at;
}

export function orderedTypes(types: readonly PageType[]): PageType[] {
    return [...types].sort((left, right) => {
        const rank = presetRank(left.preset) - presetRank(right.preset);
        if (rank !== 0) {
            return rank;
        }
        return left.name.localeCompare(right.name, undefined, { sensitivity: "base" });
    });
}

export function lastOfPreset(types: readonly PageType[], type: PageType): boolean {
    return !types.some((held) => held.id !== type.id && held.preset === type.preset);
}

export function reassignTargets(types: readonly PageType[], type: PageType): PageType[] {
    const others = types.filter((held) => held.id !== type.id);
    return [...others.filter((held) => held.preset === type.preset), ...others.filter((held) => held.preset !== type.preset)];
}

export type MetaProblem = "keyMissing" | "keyShape" | "labelMissing" | "taken";

const keyShape = /^[^\s,]+$/;

export function metaField(key: string): string {
    return `${metaPrefix}${key.trim()}`;
}

export function metaProblem(key: string, label: string, scopes: readonly FieldScope[]): MetaProblem | null {
    const trimmed = key.trim();
    if (trimmed === "") {
        return "keyMissing";
    }
    if (!keyShape.test(trimmed)) {
        return "keyShape";
    }
    if (label.trim() === "") {
        return "labelMissing";
    }
    return scopes.some((scope) => scope.field === metaField(trimmed)) ? "taken" : null;
}

export function noLinks(): LinkPolicy {
    return {
        links: false,
        rules: {
            upDepth: 0,
            downLinks: false,
            siblingMinWeight: 0,
            maxLinks: 0,
            maxPerTarget: 0,
            parentLinkWithinParagraphs: 0,
            childrenSection: false,
        },
        forbidExternal: false,
        forbidSelf: false,
        anchorStrategy: asAnchor(anchorStrategies[0]),
    };
}

export function customScope(pageTypeId: string, key: string, label: string, provider: string): SetScopeRequest {
    return {
        pageTypeId,
        field: metaField(key),
        enabled: true,
        aiWrites: false,
        instruction: "",
        minLength: 0,
        maxLength: 0,
        rewrites: false,
        links: noLinks(),
        label: label.trim(),
        provider,
    };
}
