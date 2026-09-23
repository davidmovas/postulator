import { copy } from "../../copy/index.js";
import type { JsonObject, JsonValue } from "../../domain/merge-patch.js";
import { decimal, imageSourceLabel, percent, stepLabel } from "./labels.js";
import { layered } from "./patch.js";
import type { SpecDraft } from "./spec.js";
import { isJsonObject } from "./spec.js";

const said = copy.templates.sentence;

function keysOf(held: JsonValue | undefined, all: readonly string[]): readonly string[] {
    if (isJsonObject(held)) {
        return all.filter((key) => Object.prototype.hasOwnProperty.call(held, key));
    }
    return all;
}

function joined(items: readonly string[]): string {
    if (items.length <= 1) {
        return items.join("");
    }
    return `${items.slice(0, -1).join(", ")} and ${items[items.length - 1] ?? ""}`;
}

function lengthSentences(base: SpecDraft, after: SpecDraft, held: JsonValue | undefined): string[] {
    const keys = keysOf(held, ["min", "max"]);
    if (keys.length < 2) {
        return keys.includes("min") ? [said.lengthMin(after.lengthMin)] : [said.lengthMax(after.lengthMax)];
    }
    if (after.lengthMax < base.lengthMax || (after.lengthMax === base.lengthMax && after.lengthMin > base.lengthMin)) {
        return [said.shorter(after.lengthMin, after.lengthMax)];
    }
    if (after.lengthMax > base.lengthMax || after.lengthMin < base.lengthMin) {
        return [said.longer(after.lengthMin, after.lengthMax)];
    }
    return [said.length(after.lengthMin, after.lengthMax)];
}

function keywordSentences(after: SpecDraft, held: JsonValue | undefined): string[] {
    const out: string[] = [];
    for (const key of keysOf(held, [
        "primaryInTitle",
        "primaryInH1",
        "primaryInFirstParagraph",
        "maxDensity",
    ])) {
        if (key === "primaryInTitle") {
            out.push(said.primaryInTitle(after.primaryInTitle));
        } else if (key === "primaryInH1") {
            out.push(said.primaryInH1(after.primaryInH1));
        } else if (key === "primaryInFirstParagraph") {
            out.push(said.primaryInFirstParagraph(after.primaryInFirstParagraph));
        } else {
            out.push(said.maxDensity(percent(after.maxDensity)));
        }
    }
    return out;
}

function linkSentences(after: SpecDraft, held: JsonValue | undefined): string[] {
    const out: string[] = [];
    for (const key of keysOf(held, [
        "upDepth",
        "downLinks",
        "siblingMinWeight",
        "maxLinks",
        "maxPerTarget",
        "parentLinkWithinParagraphs",
        "childrenSection",
    ])) {
        if (key === "upDepth") {
            out.push(said.upDepth(after.upDepth));
        } else if (key === "downLinks") {
            out.push(said.downLinks(after.downLinks));
        } else if (key === "siblingMinWeight") {
            out.push(said.siblingMinWeight(decimal(after.siblingMinWeight)));
        } else if (key === "maxLinks") {
            out.push(said.maxLinks(after.maxLinks));
        } else if (key === "maxPerTarget") {
            out.push(said.maxPerTarget(after.maxPerTarget));
        } else if (key === "parentLinkWithinParagraphs") {
            out.push(said.parentLinkWithinParagraphs(after.parentLinkWithinParagraphs));
        } else {
            out.push(said.childrenSection(after.childrenSection));
        }
    }
    return out;
}

function metaSentences(after: SpecDraft, held: JsonValue | undefined): string[] {
    const out: string[] = [];
    for (const key of keysOf(held, ["titlePattern", "descriptionMax"])) {
        if (key === "titlePattern") {
            out.push(after.titlePattern === "" ? said.titlePatternNone : said.titlePattern(after.titlePattern));
        } else {
            out.push(said.descriptionMax(after.descriptionMax));
        }
    }
    return out;
}

function imageSentences(after: SpecDraft, held: JsonValue | undefined): string[] {
    const out: string[] = [];
    for (const key of keysOf(held, ["featured", "inline", "source"])) {
        if (key === "featured") {
            out.push(said.featured(after.featuredImage));
        } else if (key === "inline") {
            out.push(said.inline(after.inlineImages));
        } else {
            out.push(said.imageSource(imageSourceLabel(after.imageSource)));
        }
    }
    return out;
}

function profileSentences(after: SpecDraft, held: JsonValue | undefined): string[] {
    if (!isJsonObject(held)) {
        return after.profiles.map((profile) => said.profile(profile.role, profile.provider, profile.model));
    }
    return Object.keys(held).map((role) => {
        const pinned = after.profiles.find((profile) => profile.role === role);
        return pinned === undefined ? said.profileNone(role) : said.profile(role, pinned.provider, pinned.model);
    });
}

function recipeSentences(base: SpecDraft, after: SpecDraft): string[] {
    const was = new Map(base.recipe.map((step) => [step.name, step.enabled]));
    const off: string[] = [];
    const on: string[] = [];
    for (const step of after.recipe) {
        const before = was.get(step.name) === true;
        if (before && !step.enabled) {
            off.push(stepLabel(step.name));
        }
        if (!before && step.enabled) {
            on.push(stepLabel(step.name));
        }
    }
    const out: string[] = [];
    if (off.length > 0) {
        out.push(said.stepsOff(joined(off)));
    }
    if (on.length > 0) {
        out.push(said.stepsOn(joined(on)));
    }
    out.push(...paramSentences(base, after));
    return out.length === 0 ? [said.recipe] : out;
}

function paramSentences(base: SpecDraft, after: SpecDraft): string[] {
    const before = new Map(base.recipe.map((step) => [step.name, step.params]));
    const out: string[] = [];
    for (const step of after.recipe) {
        const held = step.params;
        const was = before.get(step.name) ?? null;
        if (held === null) {
            continue;
        }
        if (step.name === "validate" && typeof held["allowErrors"] === "boolean" && was?.["allowErrors"] !== held["allowErrors"]) {
            out.push(said.allowErrors(held["allowErrors"]));
        }
        if (step.name === "repair_links" && typeof held["iterations"] === "number" && was?.["iterations"] !== held["iterations"]) {
            out.push(said.repairIterations(held["iterations"]));
        }
    }
    return out;
}

function sectionSentences(after: SpecDraft): string[] {
    if (after.sections.length === 0) {
        return [said.sectionsNone];
    }
    const headings = after.sections.map(
        (section) => (section.heading === "" ? copy.templates.sections.untitled : section.heading),
    );
    return [said.sections(after.sections.length, joined(headings))];
}

export function sentencesOf(base: SpecDraft, patch: JsonObject | null): string[] {
    if (patch === null) {
        return [];
    }
    const after = layered(base, patch);
    const out: string[] = [];
    for (const key of Object.keys(patch)) {
        const held = patch[key];
        if (key === "sections") {
            out.push(...sectionSentences(after));
        } else if (key === "tone") {
            out.push(after.tone === "" ? said.toneNone : said.tone(after.tone));
        } else if (key === "length") {
            out.push(...lengthSentences(base, after, held));
        } else if (key === "keywordRules") {
            out.push(...keywordSentences(after, held));
        } else if (key === "linkRules") {
            out.push(...linkSentences(after, held));
        } else if (key === "metaRules") {
            out.push(...metaSentences(after, held));
        } else if (key === "images") {
            out.push(...imageSentences(after, held));
        } else if (key === "modelProfiles") {
            out.push(...profileSentences(after, held));
        } else if (key === "recipe") {
            out.push(...recipeSentences(base, after));
        } else {
            out.push(said.other(key));
        }
    }
    return out;
}
