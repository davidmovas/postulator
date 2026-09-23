import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { useRun } from "../../data/hooks/runs.js";
import { absoluteTime, relativeTime, usd } from "../../domain/format.js";
import { ChevronRightIcon, Panel, PanelHeader, Skeleton, StatusBadge } from "../../ui/index.js";
import { kindLabel, statusLabel, statusTone } from "../runs/labels.js";

export interface LastRunCardProps {
    siteId: string;
    runId: string;
}

export function LastRunCard({ siteId, runId }: LastRunCardProps): ReactElement {
    const run = useRun(runId);
    const held = run.data?.run;

    return (
        <Panel>
            <PanelHeader title={copy.schedules.panel.lastRun}>
                {held === undefined ? null : (
                    <StatusBadge tone={statusTone(held.status)}>{statusLabel(held.status)}</StatusBadge>
                )}
            </PanelHeader>
            <div className="flex flex-col gap-2 p-3">
                {held === undefined ? (
                    <Skeleton height={16} width="70%" />
                ) : (
                    <>
                        <Link
                            to={`/s/${siteId}/runs/${held.id}`}
                            className="flex items-center gap-2 rounded-md border border-hairline bg-inset px-2.5 py-2 hover:bg-raised"
                        >
                            <div className="flex min-w-0 flex-1 flex-col">
                                <span className="truncate text-xs font-semibold text-ink">{kindLabel(held.kind)}</span>
                                <span
                                    className="truncate text-2xs text-ink-faint"
                                    title={absoluteTime(held.finishedAt ?? held.createdAt)}
                                >
                                    {relativeTime(held.finishedAt ?? held.createdAt)}
                                </span>
                            </div>
                            <ChevronRightIcon size={16} className="shrink-0 text-ink-dim" />
                        </Link>
                        <dl className="flex flex-wrap gap-x-4 gap-y-1 text-2xs text-ink-dim">
                            <div className="flex gap-1.5">
                                <dt>{copy.schedules.stats.items}</dt>
                                <dd className="font-mono text-ink">{held.stats.items}</dd>
                            </div>
                            <div className="flex gap-1.5">
                                <dt>{copy.schedules.stats.done}</dt>
                                <dd className="font-mono text-ink">{held.stats.done}</dd>
                            </div>
                            <div className="flex gap-1.5">
                                <dt>{copy.schedules.stats.failed}</dt>
                                <dd className={held.stats.failed > 0 ? "font-mono text-danger" : "font-mono text-ink"}>
                                    {held.stats.failed}
                                </dd>
                            </div>
                            <div className="flex gap-1.5">
                                <dt>{copy.schedules.stats.spent}</dt>
                                <dd className="font-mono text-ink">{usd(held.stats.usd)}</dd>
                            </div>
                        </dl>
                        <Link
                            to={`/s/${siteId}/runs?kind=${held.kind}`}
                            className="text-2xs text-accent hover:underline"
                        >
                            {copy.schedules.panel.allRuns}
                        </Link>
                    </>
                )}
            </div>
        </Panel>
    );
}
