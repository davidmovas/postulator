import type { ItemReport, Run, RunReport } from "../../../data/types.js";
import { terminalRunStatuses } from "../../../generated/vocab.js";
import { finalView, weigh } from "../../runs/artifacts.js";

export function finished(runs: readonly Run[]): Run[] {
    return runs.filter((run) => (terminalRunStatuses as readonly string[]).includes(run.status));
}

export interface ItemRow {
    itemId: string;
    pageId: string;
    path: string;
    status: string;
    errors: number;
    warnings: number;
    score: number | null;
    error: string;
}

const failedFirst: Readonly<Record<string, number>> = { failed: 0, cancelled: 1, paused: 2, completed: 4 };

function rank(status: string): number {
    return failedFirst[status] ?? 3;
}

export function itemRows(items: readonly ItemReport[] | null, paths: ReadonlyMap<string, string>): ItemRow[] {
    const rows = (items ?? []).map((item) => {
        const view = finalView(item.report ?? null);
        const counted = view === null ? { errors: 0, warnings: 0 } : weigh(view.findings);
        return {
            itemId: item.itemId,
            pageId: item.pageId,
            path: view?.path !== undefined && view.path !== "" ? view.path : (paths.get(item.pageId) ?? ""),
            status: item.status,
            errors: view?.errors ?? counted.errors,
            warnings: view?.warnings ?? counted.warnings,
            score: view?.score ?? null,
            error: item.error ?? "",
        };
    });
    rows.sort((left, right) => {
        const byStatus = rank(left.status) - rank(right.status);
        if (byStatus !== 0) {
            return byStatus;
        }
        if (left.errors !== right.errors) {
            return right.errors - left.errors;
        }
        return left.path.localeCompare(right.path);
    });
    return rows;
}

export interface RunTally {
    items: number;
    done: number;
    failed: number;
    usd: number;
    tokens: number;
}

export function tally(report: RunReport): RunTally {
    return {
        items: report.stats.items,
        done: report.stats.done,
        failed: report.stats.failed,
        usd: report.stats.usd,
        tokens: report.stats.tokens,
    };
}
