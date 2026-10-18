import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { useModelSummary } from "../../data/hooks/pages.js";
import type { ModelSummary } from "../../data/types.js";
import { absoluteTime, relativeTime, tokens } from "../../domain/format.js";
import { Panel, SectionLabel, Skeleton } from "../../ui/index.js";
import {
    findingCodeLabel,
    linkTypeLabel,
    parentMethodName,
    parentReasonLabel,
    rootStateLabel,
    technicalReasonLabel,
} from "../pages/model/labels.js";

const said = copy.overview.observation;

type Counts = ModelSummary["roots"];

interface Row {
    key: string;
    label: string;
    count: number;
}

export function countRows(counts: Counts, label: (key: string) => string): Row[] {
    const rows: Row[] = [];
    for (const [key, count] of Object.entries(counts ?? {})) {
        if (count !== undefined && count > 0) {
            rows.push({ key, label: label(key), count });
        }
    }
    return rows.sort((left, right) => right.count - left.count || left.label.localeCompare(right.label));
}

function Group({ title, rows }: { title: string; rows: readonly Row[] }): ReactElement {
    return (
        <section aria-label={title} className="flex min-w-0 flex-col gap-1">
            <span className="text-xs font-medium text-ink-dim">{title}</span>
            {rows.length === 0 ? (
                <span className="text-2xs text-ink-faint">{said.empty}</span>
            ) : (
                <dl className="flex flex-col gap-0.5">
                    {rows.map((row) => (
                        <div key={row.key} className="flex items-baseline justify-between gap-2 text-xs">
                            <dt className="min-w-0 truncate text-ink-soft">{row.label}</dt>
                            <dd className="shrink-0 font-mono text-ink">{tokens(row.count)}</dd>
                        </div>
                    ))}
                </dl>
            )}
        </section>
    );
}

function Stat({ label, value }: { label: string; value: string }): ReactElement {
    return (
        <span className="flex items-baseline gap-1.5 text-2xs whitespace-nowrap text-ink-dim">
            <span className="font-mono text-xs text-ink">{value}</span>
            <span>{label}</span>
        </span>
    );
}

export interface ObservationPanelProps {
    siteId: string;
}

export function ObservationPanel({ siteId }: ObservationPanelProps): ReactElement | null {
    const summary = useModelSummary(siteId);

    if (summary.isPending) {
        return (
            <Panel className="flex flex-col gap-2 p-3">
                <SectionLabel>{said.title}</SectionLabel>
                <Skeleton height={64} />
            </Panel>
        );
    }
    const held = summary.data;
    if (held === undefined) {
        return null;
    }

    return (
        <Panel className="flex flex-col gap-3 p-3">
            <div className="flex items-baseline justify-between gap-2">
                <SectionLabel>{said.title}</SectionLabel>
                <span className="truncate font-mono text-2xs text-ink-faint" title={absoluteTime(held.lastObservedAt)}>
                    {said.walks(relativeTime(held.lastObservedAt), relativeTime(held.lastFullWalkAt))}
                </span>
            </div>
            {held.observed === 0 ? (
                <p className="text-xs text-ink-dim">{said.none}</p>
            ) : (
                <>
                    <div className="flex flex-wrap gap-x-4 gap-y-1">
                        <Stat label={said.observed} value={said.observedOf(held.observed, held.pages)} />
                        <Stat label={said.rendered} value={tokens(held.rendered)} />
                        <Stat label={said.resolved} value={tokens(held.trailsResolved)} />
                        <Stat label={said.listings} value={tokens(held.listings)} />
                        <Stat label={said.readOnly} value={tokens(held.readOnly)} />
                    </div>
                    <div className="grid grid-cols-1 gap-3 @md:grid-cols-2 @xl:grid-cols-3">
                        <Group title={said.groups.roots} rows={countRows(held.roots, rootStateLabel)} />
                        <Group title={said.groups.linksByType} rows={countRows(held.linksByType, linkTypeLabel)} />
                        <Group title={said.groups.findings} rows={countRows(held.findings, findingCodeLabel)} />
                        <Group title={said.groups.trailsByMethod} rows={countRows(held.trailsByMethod, parentMethodName)} />
                        <Group title={said.groups.trailsByReason} rows={countRows(held.trailsByReason, parentReasonLabel)} />
                        <Group title={said.groups.technical} rows={countRows(held.technical, technicalReasonLabel)} />
                    </div>
                </>
            )}
        </Panel>
    );
}
