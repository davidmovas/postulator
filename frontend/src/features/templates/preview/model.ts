import type { SpecDraft } from "../spec.js";

export interface SkeletonSection {
    heading: string;
    words: number;
    required: boolean;
    share: number;
    keywordInHeading: boolean;
    imageAfter: boolean;
}

export type TitlePart = { kind: "text"; text: string } | { kind: "placeholder"; text: string };

export type LengthState = "under" | "within" | "over" | "unbounded";

export interface Skeleton {
    titleParts: TitlePart[];
    descriptionMax: number;
    titleKeyword: boolean;
    h1Keyword: boolean;
    firstParagraphKeyword: boolean;
    featured: boolean;
    inlineImages: number;
    sections: SkeletonSection[];
    parentLinkWithin: number;
    upDepth: number;
    downLinks: boolean;
    childrenSection: boolean;
    maxLinks: number;
    totalWords: number;
    lengthMin: number;
    lengthMax: number;
    lengthState: LengthState;
}

const placeholder = /\{([a-zA-Z]+)\}/g;

export function titleParts(pattern: string): TitlePart[] {
    const out: TitlePart[] = [];
    let cursor = 0;
    for (const match of pattern.matchAll(placeholder)) {
        const at = match.index ?? 0;
        if (at > cursor) {
            out.push({ kind: "text", text: pattern.slice(cursor, at) });
        }
        out.push({ kind: "placeholder", text: match[1] ?? "" });
        cursor = at + match[0].length;
    }
    if (cursor < pattern.length) {
        out.push({ kind: "text", text: pattern.slice(cursor) });
    }
    return out;
}

export function imageSlots(sectionCount: number, images: number): number[] {
    if (sectionCount === 0 || images <= 0) {
        return [];
    }
    const gaps = sectionCount - 1;
    const count = Math.min(images, gaps);
    const slots = new Set<number>();
    for (let index = 1; index <= count; index += 1) {
        slots.add(Math.floor((index * gaps) / (count + 1)));
    }
    let cursor = 0;
    while (slots.size < count) {
        slots.add(cursor);
        cursor += 1;
    }
    if (images > gaps) {
        slots.add(sectionCount - 1);
    }
    return [...slots].sort((a, b) => a - b);
}

function lengthState(total: number, min: number, max: number): LengthState {
    if (max <= 0) {
        return "unbounded";
    }
    if (total < min) {
        return "under";
    }
    return total > max ? "over" : "within";
}

export function skeletonOf(draft: SpecDraft): Skeleton {
    const totalWords = draft.sections.reduce((sum, section) => sum + Math.max(section.targetWords, 0), 0);
    const slots = new Set(imageSlots(draft.sections.length, draft.inlineImages));
    return {
        titleParts: titleParts(draft.titlePattern),
        descriptionMax: draft.descriptionMax,
        titleKeyword: draft.primaryInTitle,
        h1Keyword: draft.primaryInH1,
        firstParagraphKeyword: draft.primaryInFirstParagraph,
        featured: draft.featuredImage,
        inlineImages: draft.inlineImages,
        sections: draft.sections.map((section, index) => ({
            heading: section.heading,
            words: Math.max(section.targetWords, 0),
            required: section.required,
            share: totalWords === 0 ? 0 : Math.max(section.targetWords, 0) / totalWords,
            keywordInHeading: section.primaryInHeading,
            imageAfter: slots.has(index),
        })),
        parentLinkWithin: draft.parentLinkWithinParagraphs,
        upDepth: draft.upDepth,
        downLinks: draft.downLinks,
        childrenSection: draft.childrenSection,
        maxLinks: draft.maxLinks,
        totalWords,
        lengthMin: draft.lengthMin,
        lengthMax: draft.lengthMax,
        lengthState: lengthState(totalWords, draft.lengthMin, draft.lengthMax),
    };
}
