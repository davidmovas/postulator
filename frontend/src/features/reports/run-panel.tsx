import type { ReactElement } from "react";
import { useMemo } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { usePageTree } from "../../data/hooks/pages.js";
import { useRunReport } from "../../data/hooks/reports.js";
import { tokens, usd } from "../../domain/format.js";
import {
    Banner,
    ChevronRightIcon,
    EmptyState,
    HistoryIcon,
    Panel,
    PanelHeader,
    SkeletonRows,
    StatusBadge,
} from "../../ui/index.js";
import { kindLabel, statusLabel, statusTone } from "../runs/labels.js";
import { flattenTree } from "./model/site.js";
import { itemRows } from "./model/runs.js";

export interface RunPanelProps {
    siteId: string;
    runId: string;
}

export function RunReportPanel({ siteId, runId }: RunPanelProps): ReactElement {
    const report = useRunReport(runId === "" ? null : runId);
    const tree = usePageTree(siteId === "" ? null : siteId);
    const paths = useMemo(() => {
        const held = new Map<string, string>();
        for (const page of flattenTree(tree.data?.roots ?? null)) {
            held.set(page.id, page.path);
        }
        return held;
    }, [tree.data]);
    const rows = useMemo(() => itemRows(report.data?.items ?? null, paths), [report.data, paths]);

    if (runId === "") {
        return (
            <div className="p-3">
                <EmptyState icon={HistoryIcon} title={copy.reports.runs.choose} />
            </div>
        );
    }

    if (report.isPending) {
        return (
            <div className="p-3">
                <SkeletonRows rows={6} label={copy.reports.runs.report} />
            </div>
        );
    }

    const failure = report.error === null ? null : react(report.error);
    if (failure !== null && failure.kind !== "silent" && failure.kind !== "unlock") {
        return (
            <div className="p-3">
                <Banner tone="danger" title={failure.message} />
            </div>
        );
    }

    const held = report.data;
    if (held === undefined) {
        return (
            <div className="p-3">
                <SkeletonRows rows={6} label={copy.reports.runs.report} />
            </div>
        );
    }

    return (
        <div className="flex flex-col gap-3 p-3">
            <Panel>
                <PanelHeader title={kindLabel(held.kind)}>
                    <StatusBadge tone={statusTone(held.status)}>{statusLabel(held.status)}</StatusBadge>
                </PanelHeader>
                <div className="flex flex-col gap-2 p-3">
                    <dl className="flex flex-wrap gap-x-4 gap-y-1 text-2xs text-ink-dim">
                        <div className="flex gap-1.5">
                            <dt>{copy.reports.runs.items}</dt>
                            <dd className="font-mono text-ink">{held.stats.items}</dd>
                        </div>
                        <div className="flex gap-1.5">
                            <dt>{copy.reports.runs.outcome}</dt>
                            <dd className={held.stats.failed > 0 ? "font-mono text-danger" : "font-mono text-ink"}>
                                {held.stats.done} / {held.stats.failed}
                            </dd>
                        </div>
                        <div className="flex gap-1.5">
                            <dt>{copy.reports.runs.cost}</dt>
                            <dd className="font-mono text-ink">{usd(held.stats.usd)}</dd>
                        </div>
                        <div className="flex gap-1.5">
                            <dt>{copy.reports.runs.tokens}</dt>
                            <dd className="font-mono text-ink">{tokens(held.stats.tokens)}</dd>
                        </div>
                    </dl>
                    <Link
                        to={`/s/${siteId}/runs/${held.runId}`}
                        className="flex h-7 items-center justify-center gap-1 rounded-md border border-hairline text-sm text-ink hover:bg-raised"
                    >
                        {copy.reports.runs.open}
                        <ChevronRightIcon size={14} />
                    </Link>
                </div>
            </Panel>
            <Panel>
                <PanelHeader title={copy.reports.runs.perPage} />
                <ul className="flex flex-col">
                    {rows.map((row) => (
                        <li key={row.itemId} className="flex flex-col gap-1 border-b border-hairline p-3 last:border-b-0">
                            <div className="flex items-center gap-2">
                                <StatusBadge tone={statusTone(row.status)}>{statusLabel(row.status)}</StatusBadge>
                                <Link
                                    to={`/s/${siteId}/runs/${held.runId}/items/${row.itemId}`}
                                    title={row.path}
                                    className="min-w-0 flex-1 truncate font-mono text-xs text-accent hover:underline"
                                >
                                    {row.path}
                                </Link>
                            </div>
                            <div className="flex flex-wrap items-center gap-2 text-2xs">
                                {row.errors > 0 ? (
                                    <span className="text-danger">{copy.reports.runs.errors(row.errors)}</span>
                                ) : null}
                                {row.warnings > 0 ? (
                                    <span className="text-warn">{copy.reports.runs.warnings(row.warnings)}</span>
                                ) : null}
                                {row.errors === 0 && row.warnings === 0 ? (
                                    <span className="text-ink-faint">{copy.reports.runs.clean}</span>
                                ) : null}
                                {row.score === null ? null : (
                                    <span className="font-mono text-ink-dim">{copy.reports.runs.score(row.score)}</span>
                                )}
                            </div>
                            {row.error === "" ? null : <p className="text-2xs text-danger">{row.error}</p>}
                        </li>
                    ))}
                </ul>
            </Panel>
        </div>
    );
}
