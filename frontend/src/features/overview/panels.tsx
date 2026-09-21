import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { formatScore } from "../graph/labels.js";
import { kindLabel as runKindLabel, statusLabel as runStatusLabel } from "../runs/labels.js";
import type { EntityScore } from "../../data/types.js";
import { tokens, usd } from "../../domain/format.js";
import {
    ArrowRightAltIcon,
    BudgetGauge,
    Button,
    ChevronRightIcon,
    EmptyState,
    HubIcon,
    MonitoringIcon,
    Panel,
    ProgressBar,
    SectionLabel,
    StatusBadge,
    SyncProblemIcon,
} from "../../ui/index.js";
import type { DepthBar, EdgeTile, RunSummary } from "./model/overview.js";

export interface RunPanelProps {
    summary: RunSummary | null;
    siteId: string;
    onStart: () => void;
}

export function RunPanel({ summary, siteId, onStart }: RunPanelProps): ReactElement {
    if (summary === null) {
        return (
            <Panel className="p-3">
                <EmptyState
                    icon={MonitoringIcon}
                    title={copy.overview.run.none}
                    actions={
                        <Button variant="primary" onClick={onStart}>
                            {copy.overview.startRun}
                        </Button>
                    }
                />
            </Panel>
        );
    }
    const run = summary.run;
    return (
        <Link
            to={`/s/${siteId}/runs/${run.id}`}
            title={copy.overview.run.open}
            className="flex items-center gap-3 rounded-lg border border-hairline bg-panel px-3 py-2.5 hover:border-edge"
        >
            <div className="flex min-w-0 flex-1 flex-col gap-1">
                <div className="flex items-center gap-2">
                    <span className="truncate text-xs font-semibold text-ink">{runKindLabel(run.kind)}</span>
                    <StatusBadge tone={summary.active ? "info" : "muted"}>
                        {summary.active ? copy.overview.run.active : runStatusLabel(run.status)}
                    </StatusBadge>
                    <span className="truncate font-mono text-2xs text-ink-dim">
                        {copy.overview.run.startedBy(run.createdBy)} · {copy.overview.run.publish(run.publishMode)}
                    </span>
                </div>
                <ProgressBar
                    label={copy.overview.run.title}
                    value={summary.done + summary.failed}
                    max={Math.max(summary.total, 1)}
                    tone={summary.failed > 0 ? "warn" : summary.active ? "accent" : "ok"}
                    trailing={copy.overview.run.items(summary.done, summary.total)}
                />
            </div>
            <div className="hidden w-40 shrink-0 @md:block">
                <BudgetGauge
                    label={copy.overview.run.budget}
                    value={summary.usd}
                    max={summary.maxUsd}
                    trailing={`${usd(summary.usd)} / ${usd(summary.maxUsd)}`}
                />
            </div>
            <ChevronRightIcon size={16} className="shrink-0 text-ink-faint" />
        </Link>
    );
}

export interface DepthPanelProps {
    bars: readonly DepthBar[];
}

export function DepthPanel({ bars }: DepthPanelProps): ReactElement {
    return (
        <Panel className="flex min-h-40 flex-col gap-3 p-3">
            <div className="flex items-baseline justify-between gap-2">
                <SectionLabel>{copy.overview.depth.title}</SectionLabel>
                <span className="font-mono text-2xs text-ink-faint">{copy.overview.depth.unit}</span>
            </div>
            {bars.length === 0 ? (
                <p className="text-xs text-ink-dim">{copy.empty.depth}</p>
            ) : (
                <div className="flex min-h-0 flex-1 items-end gap-2">
                    {bars.map((bar) => (
                        <div key={bar.depth} className="flex h-full flex-1 flex-col items-center justify-end gap-1">
                            <span className="font-mono text-2xs text-ink-dim">{tokens(bar.pages)}</span>
                            <span
                                aria-hidden={true}
                                className="w-full rounded-t-sm bg-accent"
                                style={{ height: `${Math.max(2, bar.fraction * 100)}%` }}
                            />
                            <span className="font-mono text-2xs text-ink-faint">
                                {copy.overview.depth.bucket(bar.depth)}
                            </span>
                        </div>
                    ))}
                </div>
            )}
        </Panel>
    );
}

export interface TopEntitiesProps {
    entries: readonly EntityScore[];
    siteId: string;
}

export function TopEntities({ entries, siteId }: TopEntitiesProps): ReactElement {
    return (
        <Panel className="flex flex-col gap-2 p-3">
            <div className="flex items-baseline justify-between gap-2">
                <SectionLabel>{copy.overview.top.title}</SectionLabel>
                <span className="font-mono text-2xs text-ink-faint">{copy.overview.top.unit}</span>
            </div>
            {entries.length === 0 ? (
                <p className="text-xs text-ink-dim">{copy.empty.topEntities}</p>
            ) : (
                <ul className="flex flex-col gap-1">
                    {entries.map((entry) => (
                        <li key={entry.entityId} className="flex items-center gap-2">
                            <HubIcon size={13} className="shrink-0 text-ink-faint" />
                            <Link
                                to={`/s/${siteId}/graph/${entry.entityId}`}
                                className="min-w-0 flex-1 truncate text-xs text-ink-soft hover:text-accent"
                            >
                                {entry.name}
                            </Link>
                            <span
                                aria-hidden={true}
                                className="h-1 w-16 shrink-0 overflow-hidden rounded-sm bg-inset"
                            >
                                <span
                                    className="block h-full bg-accent"
                                    style={{ width: `${Math.min(100, entry.score * 100)}%` }}
                                />
                            </span>
                            <span className="w-10 shrink-0 text-right font-mono text-2xs text-ink-dim">
                                {formatScore(entry.score)}
                            </span>
                        </li>
                    ))}
                </ul>
            )}
        </Panel>
    );
}

export interface DriftPanelProps {
    count: number;
    siteId: string;
}

export function DriftPanel({ count, siteId }: DriftPanelProps): ReactElement | null {
    if (count === 0) {
        return null;
    }
    return (
        <Panel className="flex items-start gap-2 p-3">
            <SyncProblemIcon size={16} className="mt-px shrink-0 text-warn" />
            <div className="flex min-w-0 flex-1 flex-col gap-1">
                <span className="text-xs font-semibold text-warn">{copy.overview.drift.title(count)}</span>
                <span className="text-2xs text-ink-dim">{copy.overview.drift.body}</span>
                <Link
                    to={`/s/${siteId}/pages`}
                    className="inline-flex items-center gap-0.5 text-2xs font-semibold text-accent"
                >
                    {copy.overview.drift.action}
                    <ArrowRightAltIcon size={13} />
                </Link>
            </div>
        </Panel>
    );
}

export interface CoveragePanelProps {
    edges: EdgeTile;
    onRelink: () => void;
}

export function CoveragePanel({ edges, onRelink }: CoveragePanelProps): ReactElement {
    return (
        <Panel className="flex flex-col gap-2 p-3">
            <SectionLabel>{copy.overview.coverage.title}</SectionLabel>
            <ProgressBar
                label={copy.overview.coverage.title}
                value={edges.realized}
                max={Math.max(edges.approved, 1)}
                tone={edges.gap === 0 ? "ok" : "accent"}
                leading={`${tokens(edges.realized)} / ${tokens(edges.approved)}`}
                trailing={`${Math.round(edges.coverage * 100)}%`}
            />
            <span className="text-2xs text-ink-dim">
                {edges.gap === 0 ? copy.overview.coverage.closed : copy.overview.coverage.gap(edges.gap)}
            </span>
            {edges.gap === 0 ? null : (
                <Button size="sm" onClick={onRelink}>
                    {copy.overview.coverage.action}
                </Button>
            )}
        </Panel>
    );
}
