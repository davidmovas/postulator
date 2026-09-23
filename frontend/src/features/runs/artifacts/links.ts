import type { Finding, LinkClass } from "./decode.js";
import {
    boolAt,
    codeTargetMissing,
    fieldAt,
    findingsAt,
    linkClassByCode,
    listAt,
    numberAt,
    record,
    stringAt,
    stringsAt,
} from "./decode.js";

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
