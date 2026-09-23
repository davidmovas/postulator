import type { DepthBucket, EdgeTotals, EntityTotals, PageTotals, PageTreeNode, Run } from "../../../data/types.js";
import { activeRunStatuses } from "../../../generated/vocab.js";
import type { RunStatus } from "../../../generated/vocab.js";

export interface RunSummary {
    run: Run;
    active: boolean;
    done: number;
    failed: number;
    total: number;
    usd: number;
    maxUsd: number;
}

function isActiveStatus(status: string): boolean {
    return (activeRunStatuses as readonly string[]).includes(status as RunStatus);
}

export function runSummary(runs: readonly Run[]): RunSummary | null {
    const held = runs.find((run) => isActiveStatus(run.status)) ?? runs[0];
    if (held === undefined) {
        return null;
    }
    return {
        run: held,
        active: isActiveStatus(held.status),
        done: held.stats.done,
        failed: held.stats.failed,
        total: held.stats.items,
        usd: held.stats.usd,
        maxUsd: held.budget.maxUsd,
    };
}

export interface EntityTile {
    total: number;
    withCanonical: number;
    withPublished: number;
    withoutPage: number;
}

export function entityTile(totals: EntityTotals): EntityTile {
    return {
        total: totals.total,
        withCanonical: totals.withCanonicalPage,
        withPublished: totals.withPublishedPage,
        withoutPage: Math.max(0, totals.total - totals.withCanonicalPage),
    };
}

export interface StatusCount {
    status: string;
    count: number;
}

export interface PageTile {
    total: number;
    mapped: number;
    unmapped: number;
    orphans: number;
    byStatus: readonly StatusCount[];
}

export function pageTile(totals: PageTotals): PageTile {
    const byStatus: StatusCount[] = [];
    for (const [status, count] of Object.entries(totals.byStatus ?? {})) {
        if (typeof count === "number" && count > 0) {
            byStatus.push({ status, count });
        }
    }
    byStatus.sort((left, right) => right.count - left.count);
    return {
        total: totals.total,
        mapped: Math.max(0, totals.total - totals.unmapped),
        unmapped: totals.unmapped,
        orphans: totals.orphans,
        byStatus,
    };
}

export interface EdgeTile {
    approved: number;
    realized: number;
    proposed: number;
    capped: boolean;
    gap: number;
    coverage: number;
}

export function edgeTile(totals: EdgeTotals, proposed: number, capped: boolean): EdgeTile {
    const gap = Math.max(0, totals.approved - totals.realized);
    return {
        approved: totals.approved,
        realized: totals.realized,
        proposed,
        capped,
        gap,
        coverage: totals.approved === 0 ? 0 : totals.realized / totals.approved,
    };
}

export interface DepthBar {
    depth: number;
    pages: number;
    fraction: number;
}

export function histogram(buckets: readonly DepthBucket[]): DepthBar[] {
    const peak = buckets.reduce((highest, held) => Math.max(highest, held.pages), 0);
    return buckets
        .slice()
        .sort((left, right) => left.depth - right.depth)
        .map((held) => ({
            depth: held.depth,
            pages: held.pages,
            fraction: peak === 0 ? 0 : held.pages / peak,
        }));
}

export function driftCount(roots: readonly PageTreeNode[]): number {
    let count = 0;
    for (const node of roots) {
        if (node.page.drift) {
            count += 1;
        }
        count += driftCount(node.children ?? []);
    }
    return count;
}

export function isEmptySite(pages: PageTotals, entities: EntityTotals): boolean {
    return pages.total === 0 && entities.total === 0;
}
