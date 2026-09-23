import type { JsonObject, JsonValue } from "../../domain/merge-patch.js";
import { mergePatch } from "../../domain/merge-patch.js";
import type { SpecDraft } from "./spec.js";
import { draftFromJson, isJsonObject, specJsonOf } from "./spec.js";

export type Layer = "global" | "site" | "page";

export type SpecPath = readonly string[];

export const paths = {
    sections: ["sections"],
    tone: ["tone"],
    lengthMin: ["length", "min"],
    lengthMax: ["length", "max"],
    primaryInTitle: ["keywordRules", "primaryInTitle"],
    primaryInH1: ["keywordRules", "primaryInH1"],
    primaryInFirstParagraph: ["keywordRules", "primaryInFirstParagraph"],
    maxDensity: ["keywordRules", "maxDensity"],
    upDepth: ["linkRules", "upDepth"],
    downLinks: ["linkRules", "downLinks"],
    siblingMinWeight: ["linkRules", "siblingMinWeight"],
    maxLinks: ["linkRules", "maxLinks"],
    maxPerTarget: ["linkRules", "maxPerTarget"],
    parentLinkWithinParagraphs: ["linkRules", "parentLinkWithinParagraphs"],
    childrenSection: ["linkRules", "childrenSection"],
    titlePattern: ["metaRules", "titlePattern"],
    descriptionMax: ["metaRules", "descriptionMax"],
    featuredImage: ["images", "featured"],
    inlineImages: ["images", "inline"],
    imageSource: ["images", "source"],
    profiles: ["modelProfiles"],
    recipe: ["recipe"],
} as const;

export function profilePath(role: string): SpecPath {
    return ["modelProfiles", role];
}

export function patchObject(value: JsonValue | null | undefined): JsonObject | null {
    const held: JsonValue = value ?? null;
    return isJsonObject(held) ? held : null;
}

export function applyPatch(target: JsonValue, patch: JsonValue): JsonValue {
    if (!isJsonObject(patch)) {
        return patch;
    }
    const merged: JsonObject = isJsonObject(target) ? { ...target } : {};
    for (const [key, held] of Object.entries(patch)) {
        if (held === null) {
            delete merged[key];
            continue;
        }
        merged[key] = applyPatch(merged[key] ?? null, held);
    }
    return merged;
}

export function layered(base: SpecDraft, ...patches: readonly (JsonObject | null)[]): SpecDraft {
    let document: JsonValue = specJsonOf(base);
    for (const patch of patches) {
        if (patch !== null) {
            document = applyPatch(document, patch);
        }
    }
    return draftFromJson(document);
}

export function patchBetween(base: SpecDraft, edited: SpecDraft): JsonValue | undefined {
    return mergePatch(specJsonOf(base), specJsonOf(edited));
}

export function touches(patch: JsonObject | null, path: SpecPath): boolean {
    if (patch === null) {
        return false;
    }
    let cursor: JsonValue = patch;
    for (const segment of path) {
        if (!isJsonObject(cursor)) {
            return true;
        }
        if (!Object.prototype.hasOwnProperty.call(cursor, segment)) {
            return false;
        }
        cursor = cursor[segment] ?? null;
    }
    return true;
}

export function layerAt(path: SpecPath, site: JsonObject | null, page: JsonObject | null): Layer {
    if (touches(page, path)) {
        return "page";
    }
    if (touches(site, path)) {
        return "site";
    }
    return "global";
}
