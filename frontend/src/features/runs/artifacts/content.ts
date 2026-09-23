import type { Finding } from "./decode.js";
import { boolAt, findingsAt, listAt, numberAt, record, stringAt, stringsAt } from "./decode.js";

export interface JudgeView {
    score: number;
    issues: readonly string[];
    suggestions: readonly string[];
}

export function judgeView(held: unknown): JudgeView | null {
    if (record(held) === null) {
        return null;
    }
    const score = numberAt(held, "score");
    if (score === null) {
        return null;
    }
    return { score, issues: stringsAt(held, "issues"), suggestions: stringsAt(held, "suggestions") };
}

export interface PublishView {
    url: string;
    status: string;
    contentHash: string;
    wpId: number | null;
    created: boolean;
    seoApplied: readonly string[];
    skipped: readonly string[];
    findings: readonly Finding[];
}

export function publishView(held: unknown): PublishView | null {
    if (record(held) === null) {
        return null;
    }
    return {
        url: stringAt(held, "url"),
        status: stringAt(held, "status"),
        contentHash: stringAt(held, "contentHash"),
        wpId: numberAt(held, "wpId"),
        created: boolAt(held, "created"),
        seoApplied: stringsAt(held, "seoApplied"),
        skipped: stringsAt(held, "skipped"),
        findings: findingsAt(held, "findings"),
    };
}

export interface DraftSectionView {
    heading: string;
    html: string;
}

export interface DraftView {
    title: string;
    h1: string;
    summary: string;
    sections: readonly DraftSectionView[];
}

export function draftView(held: unknown): DraftView | null {
    if (record(held) === null) {
        return null;
    }
    const sections: DraftSectionView[] = [];
    for (const entry of listAt(held, "sections")) {
        if (record(entry) !== null) {
            sections.push({ heading: stringAt(entry, "heading"), html: stringAt(entry, "html") });
        }
    }
    return {
        title: stringAt(held, "title"),
        h1: stringAt(held, "h1"),
        summary: stringAt(held, "summary"),
        sections,
    };
}

export interface MetaView {
    title: string;
    description: string;
    canonical: string;
    ogTitle: string;
    ogDescription: string;
}

export function metaView(held: unknown): MetaView | null {
    if (record(held) === null) {
        return null;
    }
    return {
        title: stringAt(held, "title"),
        description: stringAt(held, "description"),
        canonical: stringAt(held, "canonical"),
        ogTitle: stringAt(held, "ogTitle"),
        ogDescription: stringAt(held, "ogDescription"),
    };
}

export interface PlacedImageView {
    role: string;
    url: string;
    alt: string;
    wpId: number | null;
}

export interface ImagesView {
    images: readonly PlacedImageView[];
    findings: readonly Finding[];
    featuredId: number | null;
}

export function imagesView(held: unknown): ImagesView | null {
    if (record(held) === null) {
        return null;
    }
    const images: PlacedImageView[] = [];
    for (const entry of listAt(held, "images")) {
        if (record(entry) !== null) {
            images.push({
                role: stringAt(entry, "role"),
                url: stringAt(entry, "url"),
                alt: stringAt(entry, "alt"),
                wpId: numberAt(entry, "wpId"),
            });
        }
    }
    return { images, findings: findingsAt(held, "findings"), featuredId: numberAt(held, "featuredId") };
}

export interface NeighbourView {
    pageId: string;
    path: string;
    outcome: string;
    anchor: string;
    detail: string;
}

export interface RelinkView {
    linked: number | null;
    conflicts: number | null;
    skipped: number | null;
    neighbours: readonly NeighbourView[];
    findings: readonly Finding[];
}

export function relinkView(held: unknown): RelinkView | null {
    if (record(held) === null) {
        return null;
    }
    const neighbours: NeighbourView[] = [];
    for (const entry of listAt(held, "neighbors")) {
        if (record(entry) !== null) {
            neighbours.push({
                pageId: stringAt(entry, "pageId"),
                path: stringAt(entry, "path"),
                outcome: stringAt(entry, "outcome"),
                anchor: stringAt(entry, "anchor"),
                detail: stringAt(entry, "detail"),
            });
        }
    }
    return {
        linked: numberAt(held, "linked"),
        conflicts: numberAt(held, "conflicts"),
        skipped: numberAt(held, "skipped"),
        neighbours,
        findings: findingsAt(held, "findings"),
    };
}

export interface SyncView {
    url: string;
    status: string;
    source: string;
    contentHash: string;
    wpId: number | null;
    links: number | null;
    modifiedAt: string;
}

export function syncView(held: unknown): SyncView | null {
    if (record(held) === null) {
        return null;
    }
    return {
        url: stringAt(held, "url"),
        status: stringAt(held, "status"),
        source: stringAt(held, "source"),
        contentHash: stringAt(held, "contentHash"),
        wpId: numberAt(held, "wpId"),
        links: numberAt(held, "links"),
        modifiedAt: stringAt(held, "modifiedAt"),
    };
}

export interface FinalView {
    pageId: string;
    path: string;
    score: number | null;
    errors: number | null;
    warnings: number | null;
    findings: readonly Finding[];
}

export function finalView(held: unknown): FinalView | null {
    if (record(held) === null) {
        return null;
    }
    return {
        pageId: stringAt(held, "pageId"),
        path: stringAt(held, "path"),
        score: numberAt(held, "score"),
        errors: numberAt(held, "errors"),
        warnings: numberAt(held, "warnings"),
        findings: findingsAt(held, "findings"),
    };
}
