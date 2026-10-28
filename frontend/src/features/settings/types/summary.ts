import { copy } from "../../../copy/index.js";
import type { FieldScope, PageType, Rule } from "../../../data/types.js";
import { fieldLabel, metaKeyOf } from "./labels.js";
import { orderedScopes, orderedTypes } from "./model.js";

const words = copy.pageTypes.writes;

const seoPair: ReadonlySet<string> = new Set(["seo_title", "seo_description"]);

const plainWords: Readonly<Record<string, string>> = {
    title: words.title,
    excerpt: words.excerpt,
    seo_canonical: words.seo_canonical,
    product_short_description: words.product_short_description,
    variation_description: words.variation_description,
    image_alt: words.image_alt,
    image_title: words.image_title,
    upsells: words.upsells,
    cross_sells: words.cross_sells,
};

function baseOf(field: string): string {
    return metaKeyOf(field) === null ? (field.split(":")[0] ?? field) : field;
}

function phraseOf(scope: FieldScope, enabled: ReadonlySet<string>): string {
    const base = baseOf(scope.field);
    switch (base) {
        case "body":
            return scope.links.links ? words.bodyLinks : words.body;
        case "product_description":
            return scope.links.links ? words.productLinks : words.product_description;
        case "seo_title":
        case "seo_description":
            return [...seoPair].every((field) => enabled.has(field)) ? words.seoPair : words[base];
        default:
            return plainWords[base] ?? fieldLabel(scope.field, scope.label);
    }
}

function capitalised(text: string): string {
    return text === "" ? text : `${text.charAt(0).toUpperCase()}${text.slice(1)}`;
}

export function writtenSummary(scopes: readonly FieldScope[]): string {
    const written = orderedScopes(scopes).filter((scope) => scope.enabled);
    const enabled = new Set(written.map((scope) => baseOf(scope.field)));
    const phrases: string[] = [];
    for (const scope of written) {
        const phrase = phraseOf(scope, enabled);
        if (!phrases.includes(phrase)) {
            phrases.push(phrase);
        }
    }
    return phrases.length === 0 ? words.nothing : `${capitalised(phrases.join(", "))}.`;
}

export interface RulesGiving {
    rules: Rule[];
    fallback: boolean;
}

export function rulesGiving(typeId: string, rules: readonly Rule[], fallback: Rule | null): RulesGiving {
    return {
        rules: rules.filter((rule) => rule.enabled && rule.typeId === typeId),
        fallback: fallback !== null && fallback.typeId === typeId,
    };
}

export interface TypeGroups {
    written: PageType[];
    never: PageType[];
}

export function typeGroups(types: readonly PageType[]): TypeGroups {
    const ordered = orderedTypes(types);
    return { written: ordered.filter((type) => type.writable), never: ordered.filter((type) => !type.writable) };
}
