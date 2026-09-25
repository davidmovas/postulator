import type { TemplateSpec } from "../../data/types.js";
import type { JsonObject, JsonValue } from "../../domain/merge-patch.js";
import { modelRoles, stepNames } from "../../generated/vocab.js";

export interface SectionDraft {
    heading: string;
    intent: string;
    targetWords: number;
    required: boolean;
    include: string[];
    primaryInHeading: boolean;
}

export interface ProfileDraft {
    role: string;
    provider: string;
    model: string;
}

export interface StepDraft {
    name: string;
    enabled: boolean;
    declared: boolean;
    params: JsonObject | null;
}

export interface SpecDraft {
    sections: SectionDraft[];
    tone: string;
    lengthMin: number;
    lengthMax: number;
    primaryInTitle: boolean;
    primaryInH1: boolean;
    primaryInFirstParagraph: boolean;
    maxDensity: number;
    upDepth: number;
    downLinks: boolean;
    siblingMinWeight: number;
    maxLinks: number;
    maxPerTarget: number;
    parentLinkWithinParagraphs: number;
    childrenSection: boolean;
    titlePattern: string;
    descriptionMax: number;
    featuredImage: boolean;
    inlineImages: number;
    imageSource: string;
    profiles: ProfileDraft[];
    recipe: StepDraft[];
}

export const imageStep = "generate_images";
const drawnSource = "ai";

export function isJsonObject(value: JsonValue | undefined): value is JsonObject {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

function branch(parent: JsonObject | null, key: string): JsonObject | null {
    if (parent === null) {
        return null;
    }
    const held = parent[key];
    return isJsonObject(held) ? held : null;
}

function text(parent: JsonObject | null, key: string): string {
    const held = parent === null ? undefined : parent[key];
    return typeof held === "string" ? held : "";
}

function count(parent: JsonObject | null, key: string): number {
    const held = parent === null ? undefined : parent[key];
    return typeof held === "number" && Number.isFinite(held) ? held : 0;
}

function flag(parent: JsonObject | null, key: string): boolean {
    const held = parent === null ? undefined : parent[key];
    return held === true;
}

function series(parent: JsonObject | null, key: string): JsonValue[] {
    const held = parent === null ? undefined : parent[key];
    return Array.isArray(held) ? held : [];
}

function words(parent: JsonObject | null, key: string): string[] {
    return series(parent, key).filter((held): held is string => typeof held === "string");
}

export function jsonOf(spec: TemplateSpec): JsonValue {
    const encoded: JsonValue = JSON.parse(JSON.stringify(spec));
    return encoded;
}

function sectionOf(value: JsonValue): SectionDraft {
    const held = isJsonObject(value) ? value : null;
    const rules = branch(held, "keywordRules");
    return {
        heading: text(held, "heading"),
        intent: text(held, "intent"),
        targetWords: count(held, "targetWords"),
        required: flag(held, "required"),
        include: words(rules, "include"),
        primaryInHeading: flag(rules, "primaryInHeading"),
    };
}

function profilesOf(held: JsonObject | null): ProfileDraft[] {
    if (held === null) {
        return [];
    }
    const known = modelRoles.filter((role) => Object.prototype.hasOwnProperty.call(held, role));
    const extra = Object.keys(held).filter((role) => !(modelRoles as readonly string[]).includes(role));
    return [...known, ...extra].map((role) => {
        const ref = branch(held, role);
        return { role, provider: text(ref, "provider"), model: text(ref, "model") };
    });
}

function recipeOf(rows: JsonValue[]): StepDraft[] {
    const params = new Map<string, JsonObject | null>();
    const enabled = new Map<string, boolean>();
    const order: string[] = [];
    for (const row of rows) {
        if (!isJsonObject(row)) {
            continue;
        }
        const name = text(row, "name");
        if (name === "" || order.includes(name)) {
            continue;
        }
        order.push(name);
        params.set(name, branch(row, "params"));
        enabled.set(name, flag(row, "enabled"));
    }
    const extra = order.filter((name) => !(stepNames as readonly string[]).includes(name));
    return [...stepNames, ...extra].map((name) => ({
        name,
        enabled: enabled.get(name) === true,
        declared: order.includes(name),
        params: params.get(name) ?? null,
    }));
}

export function draftFromJson(value: JsonValue): SpecDraft {
    const root = isJsonObject(value) ? value : null;
    const length = branch(root, "length");
    const keywords = branch(root, "keywordRules");
    const links = branch(root, "linkRules");
    const meta = branch(root, "metaRules");
    const images = branch(root, "images");
    return {
        sections: series(root, "sections").map(sectionOf),
        tone: text(root, "tone"),
        lengthMin: count(length, "min"),
        lengthMax: count(length, "max"),
        primaryInTitle: flag(keywords, "primaryInTitle"),
        primaryInH1: flag(keywords, "primaryInH1"),
        primaryInFirstParagraph: flag(keywords, "primaryInFirstParagraph"),
        maxDensity: count(keywords, "maxDensity"),
        upDepth: count(links, "upDepth"),
        downLinks: flag(links, "downLinks"),
        siblingMinWeight: count(links, "siblingMinWeight"),
        maxLinks: count(links, "maxLinks"),
        maxPerTarget: count(links, "maxPerTarget"),
        parentLinkWithinParagraphs: count(links, "parentLinkWithinParagraphs"),
        childrenSection: flag(links, "childrenSection"),
        titlePattern: text(meta, "titlePattern"),
        descriptionMax: count(meta, "descriptionMax"),
        featuredImage: flag(images, "featured"),
        inlineImages: count(images, "inline"),
        imageSource: text(images, "source"),
        profiles: profilesOf(branch(root, "modelProfiles")),
        recipe: recipeOf(series(root, "recipe")),
    };
}

export function draftOf(spec: TemplateSpec): SpecDraft {
    return draftFromJson(jsonOf(spec));
}

export function specJsonOf(draft: SpecDraft): JsonObject {
    const profiles: JsonObject = {};
    for (const profile of draft.profiles) {
        profiles[profile.role] = { provider: profile.provider, model: profile.model };
    }
    const recipe: JsonValue[] = [];
    for (const step of draft.recipe) {
        if (!step.enabled && !step.declared) {
            continue;
        }
        const row: JsonObject = { name: step.name, enabled: step.enabled };
        if (step.params !== null && Object.keys(step.params).length > 0) {
            row["params"] = step.params;
        }
        recipe.push(row);
    }
    return {
        sections: draft.sections.map((section) => ({
            heading: section.heading,
            intent: section.intent,
            targetWords: section.targetWords,
            required: section.required,
            keywordRules: { include: [...section.include], primaryInHeading: section.primaryInHeading },
        })),
        tone: draft.tone,
        length: { min: draft.lengthMin, max: draft.lengthMax },
        keywordRules: {
            primaryInTitle: draft.primaryInTitle,
            primaryInH1: draft.primaryInH1,
            primaryInFirstParagraph: draft.primaryInFirstParagraph,
            maxDensity: draft.maxDensity,
        },
        linkRules: {
            upDepth: draft.upDepth,
            downLinks: draft.downLinks,
            siblingMinWeight: draft.siblingMinWeight,
            maxLinks: draft.maxLinks,
            maxPerTarget: draft.maxPerTarget,
            parentLinkWithinParagraphs: draft.parentLinkWithinParagraphs,
            childrenSection: draft.childrenSection,
        },
        metaRules: { titlePattern: draft.titlePattern, descriptionMax: draft.descriptionMax },
        images: { featured: draft.featuredImage, inline: draft.inlineImages, source: draft.imageSource },
        modelProfiles: profiles,
        recipe,
    };
}

export function specOf(draft: SpecDraft): TemplateSpec {
    return specJsonOf(draft) as unknown as TemplateSpec;
}

export function asksForImages(draft: SpecDraft): boolean {
    return draft.featuredImage || draft.inlineImages > 0;
}

export function readyForImages(before: SpecDraft, next: SpecDraft): SpecDraft {
    if (asksForImages(before) || !asksForImages(next)) {
        return next;
    }
    const ownRecipe = next.recipe.some((step) => step.enabled || step.declared);
    return {
        ...next,
        imageSource: next.imageSource === "" ? drawnSource : next.imageSource,
        recipe: ownRecipe
            ? next.recipe.map((step) => (step.name === imageStep ? { ...step, enabled: true, declared: true } : step))
            : next.recipe,
    };
}

export function emptySection(): SectionDraft {
    return { heading: "", intent: "", targetWords: 0, required: true, include: [], primaryInHeading: false };
}

export function moved<T>(items: readonly T[], from: number, to: number): T[] {
    if (from < 0 || to < 0 || from >= items.length || to >= items.length) {
        return [...items];
    }
    const out = [...items];
    const held = out[from];
    if (held === undefined) {
        return [...items];
    }
    out.splice(from, 1);
    out.splice(to, 0, held);
    return out;
}
