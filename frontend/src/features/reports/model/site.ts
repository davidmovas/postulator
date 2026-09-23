import type { DepthBucket, Entity, EntityTotals, Page, PageAudit, PageTotals, PageTreeNode } from "../../../data/types.js";

export interface SiteTiles {
    pagesTotal: number;
    pagesMapped: number;
    mappedShare: number;
    entitiesTotal: number;
    entitiesWithPage: number;
    withPageShare: number;
    orphans: number;
    averageDepth: number;
}

function share(part: number, whole: number): number {
    return whole <= 0 ? 0 : part / whole;
}

export function averageDepth(buckets: readonly DepthBucket[]): number {
    let pages = 0;
    let weighted = 0;
    for (const bucket of buckets) {
        pages += bucket.pages;
        weighted += bucket.pages * (bucket.depth + 1);
    }
    return pages === 0 ? 0 : weighted / pages;
}

export function tiles(
    entities: EntityTotals,
    pages: PageTotals,
    depth: readonly DepthBucket[] | null,
): SiteTiles {
    const mapped = Math.max(0, pages.total - pages.unmapped);
    return {
        pagesTotal: pages.total,
        pagesMapped: mapped,
        mappedShare: share(mapped, pages.total),
        entitiesTotal: entities.total,
        entitiesWithPage: entities.withCanonicalPage,
        withPageShare: share(entities.withCanonicalPage, entities.total),
        orphans: pages.orphans,
        averageDepth: averageDepth(depth ?? []),
    };
}

export interface DepthBar {
    level: number;
    pages: number;
    fraction: number;
}

export function bars(buckets: readonly DepthBucket[] | null): DepthBar[] {
    const held = buckets ?? [];
    const peak = held.reduce((highest, bucket) => Math.max(highest, bucket.pages), 0);
    return held
        .slice()
        .sort((left, right) => left.depth - right.depth)
        .map((bucket) => ({
            level: bucket.depth + 1,
            pages: bucket.pages,
            fraction: peak === 0 ? 0 : bucket.pages / peak,
        }));
}

export function flattenTree(roots: readonly PageTreeNode[] | null): Page[] {
    const out: Page[] = [];
    const walk = (nodes: readonly PageTreeNode[]): void => {
        for (const node of nodes) {
            out.push(node.page);
            walk(node.children ?? []);
        }
    };
    walk(roots ?? []);
    return out;
}

export function driftRows(pages: readonly Page[]): Page[] {
    return pages
        .filter((page) => page.drift)
        .sort((left, right) => (right.wpModifiedAt ?? "").localeCompare(left.wpModifiedAt ?? ""));
}

export const noPage = "noPage";

export type CoverageReason = typeof noPage | string;

export interface CoverageRow {
    entityId: string;
    name: string;
    kind: string;
    path: string;
    reason: CoverageReason;
    links: number;
}

const severity: Readonly<Record<string, number>> = {
    noPage: 0,
    planned: 1,
    exists: 2,
    archived: 3,
    published: 4,
};

function rank(reason: CoverageReason): number {
    return severity[reason] ?? 2;
}

export function coverageRows(
    entities: readonly Entity[],
    pages: readonly Page[],
    audits: readonly PageAudit[],
): CoverageRow[] {
    const byId = new Map<string, Page>();
    for (const page of pages) {
        byId.set(page.id, page);
    }
    const satisfied = new Map<string, number>();
    for (const audit of audits) {
        satisfied.set(audit.pageId, audit.satisfied);
    }
    const rows = entities.map((entity) => {
        const page = entity.canonicalPageId === null ? undefined : byId.get(entity.canonicalPageId);
        return {
            entityId: entity.id,
            name: entity.name,
            kind: entity.kind,
            path: page?.path ?? "",
            reason: page === undefined ? noPage : page.status,
            links: page === undefined ? 0 : (satisfied.get(page.id) ?? 0),
        };
    });
    rows.sort((left, right) => {
        const bySeverity = rank(left.reason) - rank(right.reason);
        if (bySeverity !== 0) {
            return bySeverity;
        }
        if (left.links !== right.links) {
            return left.links - right.links;
        }
        return left.name.localeCompare(right.name);
    });
    return rows;
}

export interface AuditCard {
    audited: number;
    pages: number;
    compliant: number;
    compliantShare: number;
    missingRequired: number;
    missing: number;
    blocked: number;
    offGraph: number;
    orphans: number;
}

export function auditCard(pages: readonly PageAudit[], totals: {
    pages: number;
    audited: number;
    required: number;
    satisfied: number;
    missing: number;
    missingRequired: number;
    blocked: number;
    offGraph: number;
    orphans: number;
}): AuditCard {
    const audited = pages.filter((page) => page.skipReason === "");
    const compliant = audited.filter((page) => page.missing === 0 && page.blocked === 0).length;
    return {
        audited: totals.audited,
        pages: totals.pages,
        compliant,
        compliantShare: share(compliant, audited.length),
        missingRequired: totals.missingRequired,
        missing: totals.missing,
        blocked: totals.blocked,
        offGraph: totals.offGraph,
        orphans: totals.orphans,
    };
}
