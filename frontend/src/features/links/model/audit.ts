import type { PageAudit } from "../../../data/types.js";
import type { GraphIndex } from "../../graph/model/index.js";
import type { LinksQuery, Show } from "./params.js";
import { shows } from "./params.js";

export type Severity = "danger" | "warn" | "ok" | "muted";

const rank: Readonly<Record<Severity, number>> = { danger: 0, warn: 1, ok: 2, muted: 3 };

export function severityOf(row: PageAudit): Severity {
    if (row.skipReason !== "") {
        return "muted";
    }
    if (row.missingRequired > 0 || row.blocked > 0) {
        return "danger";
    }
    return row.missing > 0 ? "warn" : "ok";
}

export function hasProblem(row: PageAudit): boolean {
    return row.missing > 0 || row.blocked > 0 || row.offGraph > 0 || row.orphan;
}

function shown(row: PageAudit, show: Show): boolean {
    switch (show) {
        case "all":
            return true;
        case "missing":
            return row.missing > 0;
        case "missingRequired":
            return row.missingRequired > 0;
        case "blocked":
            return row.blocked > 0;
        case "offGraph":
            return row.offGraph > 0;
        case "orphans":
            return row.orphan;
        case "skipped":
            return row.skipReason !== "";
    }
}

function subtreeOf(index: GraphIndex, entityId: string): ReadonlySet<string> {
    const kept = new Set<string>([entityId]);
    const stack = [entityId];
    while (stack.length > 0) {
        const current = stack.pop();
        if (current === undefined) {
            break;
        }
        for (const child of index.children.get(current) ?? []) {
            if (!kept.has(child)) {
                kept.add(child);
                stack.push(child);
            }
        }
    }
    return kept;
}

function compareBySeverity(left: PageAudit, right: PageAudit): number {
    const leftProblem = hasProblem(left) ? 0 : 1;
    const rightProblem = hasProblem(right) ? 0 : 1;
    if (leftProblem !== rightProblem) {
        return leftProblem - rightProblem;
    }
    const bySeverity = rank[severityOf(left)] - rank[severityOf(right)];
    if (bySeverity !== 0) {
        return bySeverity;
    }
    for (const key of ["missingRequired", "blocked", "missing"] as const) {
        if (left[key] !== right[key]) {
            return right[key] - left[key];
        }
    }
    if (left.orphan !== right.orphan) {
        return left.orphan ? -1 : 1;
    }
    if (left.offGraph !== right.offGraph) {
        return right.offGraph - left.offGraph;
    }
    return left.path < right.path ? -1 : left.path > right.path ? 1 : 0;
}

export function rows(pages: readonly PageAudit[], query: LinksQuery, index: GraphIndex | null): PageAudit[] {
    const subtree = query.entity === "" ? null : index === null ? new Set([query.entity]) : subtreeOf(index, query.entity);
    const kept = pages.filter(
        (row) =>
            shown(row, query.show) &&
            (query.status === "" || row.status === query.status) &&
            (subtree === null || (row.entityId !== "" && subtree.has(row.entityId))),
    );
    kept.sort(query.sort === "path" ? (left, right) => (left.path < right.path ? -1 : left.path > right.path ? 1 : 0) : compareBySeverity);
    return kept;
}

export function showCounts(pages: readonly PageAudit[]): Record<Show, number> {
    const counts = Object.fromEntries(shows.map((show) => [show, 0])) as Record<Show, number>;
    for (const row of pages) {
        for (const show of shows) {
            if (shown(row, show)) {
                counts[show] += 1;
            }
        }
    }
    return counts;
}

export function tintByEntity(pages: readonly PageAudit[]): ReadonlyMap<string, Severity> {
    const tint = new Map<string, Severity>();
    for (const row of pages) {
        if (row.entityId === "") {
            continue;
        }
        const severity = severityOf(row);
        const held = tint.get(row.entityId);
        if (held === undefined || rank[severity] < rank[held]) {
            tint.set(row.entityId, severity);
        }
    }
    return tint;
}
