import { copy } from "../../../copy/index.js";
import type { ClassificationGroup, ClassificationGroupKey, RuleConditions, RuleURL } from "../../../data/types.js";
import type { SchemaType, URLMode } from "../../../generated/vocab.js";
import { isOneOf, pageWpTypes, schemaTypes, urlModes } from "../../../generated/vocab.js";
import { postTypeWord, schemaWord } from "./labels.js";

const said = copy.classification.sentence;

export type TriState = "any" | "yes" | "no";

export type UnderKind = "none" | "page" | "type";

export type ListingKind = "any" | "present" | "absent";

export interface UrlDraft {
    mode: URLMode;
    value: string;
}

export interface ConditionsDraft {
    postTypes: readonly string[];
    urls: readonly UrlDraft[];
    slugs: readonly string[];
    under: UnderKind;
    underPage: string;
    underType: string;
    levelMin: string;
    levelMax: string;
    listing: ListingKind;
    minProducts: string;
    schema: readonly SchemaType[];
    wordsMin: string;
    wordsMax: string;
    minH2: string;
    technical: TriState;
    frontPage: TriState;
    shop: TriState;
}

export const blankConditions: ConditionsDraft = Object.freeze({
    postTypes: [],
    urls: [],
    slugs: [],
    under: "none",
    underPage: "",
    underType: "",
    levelMin: "",
    levelMax: "",
    listing: "any",
    minProducts: "",
    schema: [],
    wordsMin: "",
    wordsMax: "",
    minH2: "",
    technical: "any",
    frontPage: "any",
    shop: "any",
});

export const knownPostTypes: readonly string[] = pageWpTypes;

function text(value: number | null | undefined): string {
    return value === null || value === undefined ? "" : String(value);
}

function tri(value: boolean | null | undefined): TriState {
    if (value === null || value === undefined) {
        return "any";
    }
    return value ? "yes" : "no";
}

export function draftOf(conditions: RuleConditions | null | undefined): ConditionsDraft {
    if (conditions === null || conditions === undefined) {
        return blankConditions;
    }
    const ancestor = conditions.ancestor ?? null;
    const listing = conditions.listing ?? null;
    return {
        postTypes: [...(conditions.postTypes ?? [])],
        urls: (conditions.urls ?? []).map((url) => ({
            mode: isOneOf(urlModes, url.mode) ? url.mode : "prefix",
            value: url.value,
        })),
        slugs: [...(conditions.slugs ?? [])],
        under: ancestor === null ? "none" : (ancestor.pageId ?? "") !== "" ? "page" : (ancestor.typeId ?? "") !== "" ? "type" : "none",
        underPage: ancestor?.pageId ?? "",
        underType: ancestor?.typeId ?? "",
        levelMin: text(conditions.level?.min),
        levelMax: text(conditions.level?.max),
        listing: listing === null ? "any" : listing.present ? "present" : "absent",
        minProducts: listing === null || (listing.minProducts ?? 0) === 0 ? "" : String(listing.minProducts),
        schema: (conditions.schema ?? []).map((held): string => held).filter((held): held is SchemaType => isOneOf(schemaTypes, held)),
        wordsMin: text(conditions.words?.min),
        wordsMax: text(conditions.words?.max),
        minH2: (conditions.minH2 ?? 0) === 0 ? "" : String(conditions.minH2),
        technical: tri(conditions.technical),
        frontPage: tri(conditions.frontPage),
        shop: tri(conditions.shop),
    };
}

export function numberOf(raw: string): number | undefined {
    const trimmed = raw.trim();
    if (trimmed === "") {
        return undefined;
    }
    const parsed = Number(trimmed);
    return Number.isFinite(parsed) ? Math.trunc(parsed) : undefined;
}

function boolOf(state: TriState): boolean | undefined {
    if (state === "any") {
        return undefined;
    }
    return state === "yes";
}

function cleaned(values: readonly string[]): string[] {
    return [...new Set(values.map((held) => held.trim()).filter((held) => held !== ""))];
}

function range(min: string, max: string): { min?: number; max?: number } | undefined {
    const low = numberOf(min);
    const high = numberOf(max);
    if (low === undefined && high === undefined) {
        return undefined;
    }
    return { ...(low === undefined ? {} : { min: low }), ...(high === undefined ? {} : { max: high }) };
}

export function conditionsOf(draft: ConditionsDraft): RuleConditions {
    const out: RuleConditions = {};
    const postTypes = cleaned(draft.postTypes);
    if (postTypes.length > 0) {
        out.postTypes = postTypes;
    }
    const urls = draft.urls
        .map((url) => ({ mode: url.mode, value: url.value.trim() }))
        .filter((url) => url.value !== "")
        .map((url) => ({ mode: url.mode as RuleURL["mode"], value: url.value }));
    if (urls.length > 0) {
        out.urls = urls;
    }
    const slugs = cleaned(draft.slugs);
    if (slugs.length > 0) {
        out.slugs = slugs;
    }
    if (draft.under === "page" && draft.underPage !== "") {
        out.ancestor = { pageId: draft.underPage };
    }
    if (draft.under === "type" && draft.underType !== "") {
        out.ancestor = { typeId: draft.underType };
    }
    const level = range(draft.levelMin, draft.levelMax);
    if (level !== undefined) {
        out.level = level;
    }
    if (draft.listing !== "any") {
        const least = numberOf(draft.minProducts);
        out.listing =
            draft.listing === "present" && least !== undefined && least > 0 ? { present: true, minProducts: least } : { present: draft.listing === "present" };
    }
    if (draft.schema.length > 0) {
        out.schema = draft.schema.map((held) => held as NonNullable<RuleConditions["schema"]>[number]);
    }
    const words = range(draft.wordsMin, draft.wordsMax);
    if (words !== undefined) {
        out.words = words;
    }
    const minH2 = numberOf(draft.minH2);
    if (minH2 !== undefined && minH2 > 0) {
        out.minH2 = minH2;
    }
    const flags = { technical: boolOf(draft.technical), frontPage: boolOf(draft.frontPage), shop: boolOf(draft.shop) };
    for (const [key, value] of Object.entries(flags)) {
        if (value !== undefined) {
            Object.assign(out, { [key]: value });
        }
    }
    return out;
}

export function asksAnything(conditions: RuleConditions): boolean {
    return Object.keys(conditions).length > 0;
}

export interface Names {
    type: (typeId: string) => string;
    page: (pageId: string) => string;
}

function urlPhrases(urls: readonly RuleURL[]): string[] {
    const byMode = new Map<URLMode, string[]>();
    for (const url of urls) {
        const mode: URLMode = isOneOf(urlModes, url.mode) ? url.mode : "prefix";
        const held = byMode.get(mode);
        if (held === undefined) {
            byMode.set(mode, [url.value]);
        } else {
            held.push(url.value);
        }
    }
    return [...byMode].map(([mode, values]) => said.url[mode](values));
}

function rangePhrase(
    held: { min?: number | null; max?: number | null } | null | undefined,
    words: { exactly?: (value: number) => string; from: (value: number) => string; to: (value: number) => string; between: (min: number, max: number) => string },
): string | null {
    const min = held?.min ?? null;
    const max = held?.max ?? null;
    if (min === null && max === null) {
        return null;
    }
    if (min !== null && max !== null) {
        return min === max && words.exactly !== undefined ? words.exactly(min) : words.between(min, max);
    }
    return min === null ? words.to(max ?? 0) : words.from(min);
}

function flagPhrase(value: boolean | null | undefined, yes: string, no: string): string | null {
    if (value === null || value === undefined) {
        return null;
    }
    return value ? yes : no;
}

export function sentence(conditions: RuleConditions, names: Names): string {
    const parts: (string | null)[] = [
        (conditions.postTypes ?? []).length === 0 ? null : said.postTypes((conditions.postTypes ?? []).map(postTypeWord)),
        (conditions.urls ?? []).length === 0 ? null : said.urls(urlPhrases(conditions.urls ?? [])),
        (conditions.slugs ?? []).length === 0 ? null : said.slugs(conditions.slugs ?? []),
        (conditions.ancestor?.pageId ?? "") !== ""
            ? said.underPage(names.page(conditions.ancestor?.pageId ?? ""))
            : (conditions.ancestor?.typeId ?? "") !== ""
              ? said.underType(names.type(conditions.ancestor?.typeId ?? ""))
              : null,
        rangePhrase(conditions.level, { exactly: said.levelExactly, from: said.levelFrom, to: said.levelTo, between: said.levelBetween }),
        conditions.listing === null || conditions.listing === undefined
            ? null
            : !conditions.listing.present
              ? said.noListing
              : (conditions.listing.minProducts ?? 0) > 0
                ? said.listingAtLeast(conditions.listing.minProducts ?? 0)
                : said.listing,
        (conditions.schema ?? []).length === 0 ? null : said.schema((conditions.schema ?? []).map((held) => schemaWord(held))),
        rangePhrase(conditions.words, { from: said.wordsFrom, to: said.wordsTo, between: said.wordsBetween }),
        (conditions.minH2 ?? 0) > 0 ? said.minH2(conditions.minH2 ?? 0) : null,
        flagPhrase(conditions.technical, said.technical, said.notTechnical),
        flagPhrase(conditions.frontPage, said.frontPage, said.notFrontPage),
        flagPhrase(conditions.shop, said.shop, said.notShop),
    ];
    const held = parts.filter((part): part is string => part !== null);
    return held.length === 0 ? said.every : said.lead(held);
}

export function reordered(ids: readonly string[], id: string, delta: -1 | 1): string[] | null {
    const at = ids.indexOf(id);
    const to = at + delta;
    if (at < 0 || to < 0 || to >= ids.length) {
        return null;
    }
    const next = [...ids];
    const moving = next[at];
    const other = next[to];
    if (moving === undefined || other === undefined) {
        return null;
    }
    next[at] = other;
    next[to] = moving;
    return next;
}

export function groupKey(group: ClassificationGroupKey): string {
    return `${group.from}\u0000${group.to}`;
}

export function keyOf(key: string): ClassificationGroupKey {
    const at = key.indexOf("\u0000");
    return { from: key.slice(0, at), to: key.slice(at + 1) };
}

export function closedOver(
    chosen: ReadonlySet<string>,
    groups: readonly ClassificationGroup[],
    key: string,
    on: boolean,
): ReadonlySet<string> {
    const byKey = new Map(groups.map((group) => [groupKey(group), group]));
    const next = new Set(chosen);
    if (on) {
        const stack = [key];
        while (stack.length > 0) {
            const held = stack.pop();
            if (held === undefined || next.has(held) || !byKey.has(held)) {
                continue;
            }
            next.add(held);
            for (const needed of byKey.get(held)?.dependsOn ?? []) {
                stack.push(groupKey(needed));
            }
        }
        return next;
    }
    const removed = new Set([key]);
    next.delete(key);
    for (let changed = true; changed; ) {
        changed = false;
        for (const group of groups) {
            const held = groupKey(group);
            if (next.has(held) && (group.dependsOn ?? []).some((needed) => removed.has(groupKey(needed)))) {
                next.delete(held);
                removed.add(held);
                changed = true;
            }
        }
    }
    return next;
}

export function pagesIn(groups: readonly ClassificationGroup[], chosen: ReadonlySet<string> | null): number {
    return groups.filter((group) => chosen === null || chosen.has(groupKey(group))).reduce((sum, group) => sum + group.pages, 0);
}

export interface FieldPlace {
    condition: string;
    index: number | null;
}

const fieldPattern = /^conditions\.([A-Za-z0-9]+)(?:\[(\d+)\])?/;

export function placeOf(field: string | null): FieldPlace | null {
    if (field === null) {
        return null;
    }
    const matched = fieldPattern.exec(field);
    if (matched === null) {
        return null;
    }
    const index = matched[2];
    return { condition: matched[1] ?? "", index: index === undefined ? null : Number(index) };
}
