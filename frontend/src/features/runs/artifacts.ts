export type Severity = "error" | "warn" | "info";

export const severityError: Severity = "error";
export const severityWarn: Severity = "warn";

export type LinkClass = "graph" | "self" | "external" | "unknown_internal";

export const codeTargetMissing = "target_missing";

const linkClassByCode: Readonly<Record<string, LinkClass>> = {
    self_link: "self",
    external_link: "external",
    unknown_internal_link: "unknown_internal",
};

export const linkClasses: readonly LinkClass[] = ["graph", "self", "external", "unknown_internal"];

export function decodeArtifact(content: string): unknown {
    if (content === "") {
        return null;
    }
    try {
        return JSON.parse(content) as unknown;
    } catch {
        return null;
    }
}

function record(held: unknown): Record<string, unknown> | null {
    if (typeof held !== "object" || held === null || Array.isArray(held)) {
        return null;
    }
    return held as Record<string, unknown>;
}

function stringAt(held: unknown, key: string): string {
    const source = record(held);
    const value = source === null ? undefined : source[key];
    return typeof value === "string" ? value : "";
}

function numberAt(held: unknown, key: string): number | null {
    const source = record(held);
    const value = source === null ? undefined : source[key];
    return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function boolAt(held: unknown, key: string): boolean {
    const source = record(held);
    return source !== null && source[key] === true;
}

function listAt(held: unknown, key: string): readonly unknown[] {
    const source = record(held);
    const value = source === null ? undefined : source[key];
    return Array.isArray(value) ? (value as readonly unknown[]) : [];
}

function stringsAt(held: unknown, key: string): readonly string[] {
    const out: string[] = [];
    for (const entry of listAt(held, key)) {
        if (typeof entry === "string" && entry !== "") {
            out.push(entry);
        }
    }
    return out;
}

function fieldAt(held: unknown, key: string): unknown {
    const source = record(held);
    return source === null ? null : source[key];
}

export interface Finding {
    severity: Severity;
    code: string;
    message: string;
    details: Record<string, unknown> | null;
}

function severityOf(held: unknown): Severity {
    const raw = stringAt(held, "severity");
    if (raw === severityError || raw === severityWarn) {
        return raw;
    }
    return "info";
}

export function findingsAt(held: unknown, key: string): readonly Finding[] {
    const out: Finding[] = [];
    for (const entry of listAt(held, key)) {
        if (record(entry) === null) {
            continue;
        }
        out.push({
            severity: severityOf(entry),
            code: stringAt(entry, "code"),
            message: stringAt(entry, "message"),
            details: record(fieldAt(entry, "details")),
        });
    }
    return out;
}

export function weigh(findings: readonly Finding[]): { errors: number; warnings: number } {
    let errors = 0;
    let warnings = 0;
    for (const finding of findings) {
        if (finding.severity === severityError) {
            errors += 1;
        } else if (finding.severity === severityWarn) {
            warnings += 1;
        }
    }
    return { errors, warnings };
}

export interface LinkTargetView {
    entityId: string;
    pageId: string;
    url: string;
    anchors: readonly string[];
    relation: string;
    required: boolean;
}

function targetOf(held: unknown): LinkTargetView {
    return {
        entityId: stringAt(held, "entityId"),
        pageId: stringAt(held, "pageId"),
        url: stringAt(held, "url"),
        anchors: stringsAt(held, "anchors"),
        relation: stringAt(held, "relation"),
        required: boolAt(held, "required"),
    };
}

function targetsAt(held: unknown, key: string): readonly LinkTargetView[] {
    const out: LinkTargetView[] = [];
    for (const entry of listAt(held, key)) {
        if (record(entry) !== null) {
            out.push(targetOf(entry));
        }
    }
    return out;
}

export interface LinkContextView {
    pageId: string;
    pageUrl: string;
    entityId: string;
    targets: readonly LinkTargetView[];
}

export function linkContextView(held: unknown): LinkContextView | null {
    if (record(held) === null) {
        return null;
    }
    return {
        pageId: stringAt(held, "pageId"),
        pageUrl: stringAt(held, "pageUrl"),
        entityId: stringAt(held, "entityId"),
        targets: targetsAt(held, "targets"),
    };
}

export interface PlacementView {
    target: LinkTargetView;
    anchor: string;
    paragraphIndex: number | null;
}

export interface ValidationView {
    pageId: string;
    score: number | null;
    complianceScore: number | null;
    structureScore: number | null;
    compliance: readonly Finding[];
    structure: readonly Finding[];
    placed: readonly PlacementView[];
    missing: readonly LinkTargetView[];
}

export function validationView(held: unknown): ValidationView | null {
    if (record(held) === null) {
        return null;
    }
    const compliance = fieldAt(held, "compliance");
    const structure = fieldAt(held, "structure");
    const links = fieldAt(held, "links");
    const placed: PlacementView[] = [];
    for (const entry of listAt(links, "placed")) {
        if (record(entry) === null) {
            continue;
        }
        placed.push({
            target: targetOf(fieldAt(entry, "target")),
            anchor: stringAt(entry, "anchor"),
            paragraphIndex: numberAt(entry, "paragraphIndex"),
        });
    }
    return {
        pageId: stringAt(held, "pageId"),
        score: numberAt(held, "score"),
        complianceScore: numberAt(compliance, "score"),
        structureScore: numberAt(structure, "score"),
        compliance: findingsAt(compliance, "items"),
        structure: findingsAt(structure, "items"),
        placed,
        missing: targetsAt(links, "missing"),
    };
}

export interface ActualLink {
    href: string;
    anchor: string;
    kind: LinkClass;
}

export function actualLinks(view: ValidationView): readonly ActualLink[] {
    const out: ActualLink[] = [];
    for (const placement of view.placed) {
        out.push({ href: placement.target.url, anchor: placement.anchor, kind: "graph" });
    }
    for (const finding of view.compliance) {
        const kind = linkClassByCode[finding.code];
        if (kind === undefined) {
            continue;
        }
        const href = finding.details === null ? "" : finding.details["href"];
        out.push({ href: typeof href === "string" ? href : "", anchor: "", kind });
    }
    return out;
}

export function countByClass(links: readonly ActualLink[]): Readonly<Record<LinkClass, number>> {
    const counts: Record<LinkClass, number> = { graph: 0, self: 0, external: 0, unknown_internal: 0 };
    for (const link of links) {
        counts[link.kind] += 1;
    }
    return counts;
}

export interface OwedTarget {
    target: LinkTargetView;
    satisfied: boolean;
    anchor: string;
}

export function owedTargets(
    context: LinkContextView | null,
    validation: ValidationView | null,
): readonly OwedTarget[] {
    const declared = context === null ? [] : context.targets;
    const fallback = validation === null ? [] : validation.placed.map((placement) => placement.target);
    const missing = validation === null ? [] : validation.missing;
    const source = declared.length > 0 ? declared : [...fallback, ...missing];

    const anchors = new Map<string, string>();
    if (validation !== null) {
        for (const placement of validation.placed) {
            anchors.set(placement.target.url, placement.anchor);
        }
    }
    const absent = new Set<string>();
    if (validation !== null) {
        for (const target of validation.missing) {
            absent.add(target.url);
        }
        for (const finding of validation.compliance) {
            if (finding.code !== codeTargetMissing || finding.details === null) {
                continue;
            }
            const pageId = finding.details["targetPageId"];
            if (typeof pageId === "string" && pageId !== "") {
                absent.add(pageId);
            }
        }
    }

    const out: OwedTarget[] = [];
    const seen = new Set<string>();
    for (const target of source) {
        const key = target.url === "" ? target.pageId : target.url;
        if (seen.has(key)) {
            continue;
        }
        seen.add(key);
        const placed = anchors.get(target.url);
        const satisfied = placed !== undefined && !absent.has(target.url) && !absent.has(target.pageId);
        out.push({ target, satisfied, anchor: placed ?? "" });
    }
    return out;
}

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
    skipped: readonly string[];
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
    return { images, skipped: stringsAt(held, "skipped"), featuredId: numberAt(held, "featuredId") };
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
