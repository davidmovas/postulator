import { failure, fieldOf } from "../../data/errors.js";
import type { RunBrief, RunDirective } from "../../data/types.js";

export const briefLines = ["goal", "focus", "tone", "avoid"] as const;
export type BriefLine = (typeof briefLines)[number];

export const briefTexts = ["reference", "styleSample"] as const;
export type BriefText = (typeof briefTexts)[number];

export type BriefKey = BriefLine | BriefText;

export type BriefDraft = Readonly<Record<BriefKey, string>>;

export const limits = {
    line: 1000,
    reference: 20000,
    styleSample: 4000,
    notes: 2000,
    sections: 10,
    listed: 20,
    item: 200,
    mustLinks: 20,
    words: 20000,
} as const;

export const emptyBrief: BriefDraft = { goal: "", focus: "", tone: "", avoid: "", reference: "", styleSample: "" };

export interface MustLinkDraft {
    pageId: string;
    path: string;
    anchor: string;
}

export interface DirectiveDraft {
    notes: string;
    addSections: readonly string[];
    keywords: readonly string[];
    anchors: readonly string[];
    mustLink: readonly MustLinkDraft[];
    minWords: string;
    maxWords: string;
}

export const emptyDirective: DirectiveDraft = {
    notes: "",
    addSections: [],
    keywords: [],
    anchors: [],
    mustLink: [],
    minWords: "",
    maxWords: "",
};

export type Directives = ReadonlyMap<string, DirectiveDraft>;

export function runes(text: string): number {
    return [...text].length;
}

export function limitOf(key: BriefKey): number {
    switch (key) {
        case "reference":
            return limits.reference;
        case "styleSample":
            return limits.styleSample;
        default:
            return limits.line;
    }
}

export function filledOf(draft: BriefDraft): number {
    return [...briefLines, ...briefTexts].filter((key) => draft[key].trim() !== "").length;
}

export function briefOf(draft: BriefDraft): RunBrief | undefined {
    const out: RunBrief = {};
    for (const key of [...briefLines, ...briefTexts]) {
        const held = draft[key].trim();
        if (held !== "") {
            out[key] = held;
        }
    }
    return Object.keys(out).length === 0 ? undefined : out;
}

export function wordsOf(typed: string): number {
    const parsed = Number.parseInt(typed.trim(), 10);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

function texts(values: readonly string[]): string[] {
    return values.map((value) => value.trim()).filter((value) => value !== "");
}

export function directiveOf(pageId: string, draft: DirectiveDraft | undefined): RunDirective | null {
    if (draft === undefined) {
        return null;
    }
    const out: RunDirective = { pageId };
    const notes = draft.notes.trim();
    if (notes !== "") {
        out.notes = notes;
    }
    const sections = texts(draft.addSections);
    if (sections.length > 0) {
        out.addSections = sections;
    }
    const keywords = texts(draft.keywords);
    if (keywords.length > 0) {
        out.keywords = keywords;
    }
    const anchors = texts(draft.anchors);
    if (anchors.length > 0) {
        out.anchors = anchors;
    }
    if (draft.mustLink.length > 0) {
        out.mustLink = draft.mustLink.map((link) => {
            const anchor = link.anchor.trim();
            return anchor === "" ? { pageId: link.pageId } : { pageId: link.pageId, anchor };
        });
    }
    const least = wordsOf(draft.minWords);
    const most = wordsOf(draft.maxWords);
    if (least > 0) {
        out.minWords = least;
    }
    if (most > 0) {
        out.maxWords = most;
    }
    return Object.keys(out).length === 1 ? null : out;
}

export function directivesOf(drafts: Directives, pageIds: readonly string[]): RunDirective[] {
    const out: RunDirective[] = [];
    for (const pageId of pageIds) {
        const held = directiveOf(pageId, drafts.get(pageId));
        if (held !== null) {
            out.push(held);
        }
    }
    return out;
}

export type DirectiveProblem =
    "notes" | "addSections" | "keywords" | "anchors" | "mustLink" | "minWords" | "maxWords" | "wordsOrder";

function listProblem(values: readonly string[], most: number): boolean {
    return values.length > most || values.some((value) => runes(value) > limits.item);
}

function wordsProblem(typed: string): boolean {
    const trimmed = typed.trim();
    if (trimmed === "") {
        return false;
    }
    const parsed = Number(trimmed);
    return !Number.isInteger(parsed) || parsed < 0 || parsed > limits.words;
}

export function directiveProblems(draft: DirectiveDraft): Set<DirectiveProblem> {
    const out = new Set<DirectiveProblem>();
    if (runes(draft.notes) > limits.notes) {
        out.add("notes");
    }
    if (listProblem(draft.addSections, limits.sections)) {
        out.add("addSections");
    }
    if (listProblem(draft.keywords, limits.listed)) {
        out.add("keywords");
    }
    if (listProblem(draft.anchors, limits.listed)) {
        out.add("anchors");
    }
    if (draft.mustLink.length > limits.mustLinks || draft.mustLink.some((link) => runes(link.anchor) > limits.item)) {
        out.add("mustLink");
    }
    if (wordsProblem(draft.minWords)) {
        out.add("minWords");
    }
    const least = wordsOf(draft.minWords);
    const most = wordsOf(draft.maxWords);
    if (wordsProblem(draft.maxWords)) {
        out.add("maxWords");
    } else if (most > 0 && most < least) {
        out.add("wordsOrder");
    }
    return out;
}

export function briefProblems(draft: BriefDraft): Set<BriefKey> {
    return new Set([...briefLines, ...briefTexts].filter((key) => runes(draft[key]) > limitOf(key)));
}

export function blockedByDrafts(brief: BriefDraft, drafts: Directives, pageIds: readonly string[]): boolean {
    if (briefProblems(brief).size > 0) {
        return true;
    }
    return pageIds.some((pageId) => {
        const held = drafts.get(pageId);
        return held !== undefined && directiveProblems(held).size > 0;
    });
}

export function directiveSize(draft: DirectiveDraft | undefined): number {
    if (draft === undefined) {
        return 0;
    }
    return (
        (draft.notes.trim() === "" ? 0 : 1) +
        texts(draft.addSections).length +
        texts(draft.keywords).length +
        texts(draft.anchors).length +
        draft.mustLink.length +
        (wordsOf(draft.minWords) > 0 ? 1 : 0) +
        (wordsOf(draft.maxWords) > 0 ? 1 : 0)
    );
}

export const mustLinkField = "directives.mustLink";

export interface MustLinkRefusal {
    message: string;
    from: string;
    to: string;
}

export function mustLinkRefusal(thrown: unknown): MustLinkRefusal | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reported = failure(thrown);
    if (reported.code !== "INVALID" || fieldOf(reported) !== mustLinkField) {
        return null;
    }
    const from = reported.details?.["from"];
    const to = reported.details?.["to"];
    return {
        message: reported.message,
        from: typeof from === "string" ? from : "",
        to: typeof to === "string" ? to : "",
    };
}
