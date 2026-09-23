import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { EmptyState, MonitoringIcon, Panel, PanelHeader, cx } from "../../ui/index.js";
import { shareOf, shareTone } from "./labels.js";
import type { AuditCard, DepthBar } from "./model/site.js";
import { toneClasses } from "../../ui/index.js";

const barTrack = 112;

export interface HistogramProps {
    bars: readonly DepthBar[];
}

export function Histogram({ bars }: HistogramProps): ReactElement {
    return (
        <Panel className="min-w-0 flex-1">
            <PanelHeader title={copy.reports.depth.title} />
            {bars.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={MonitoringIcon} title={copy.empty.depth} />
                </div>
            ) : (
                <ul className="flex items-end gap-2 p-3">
                    {bars.map((bar) => (
                        <li
                            key={bar.level}
                            className="flex min-w-0 flex-1 flex-col items-center gap-1"
                            title={copy.reports.depth.pages(bar.pages)}
                        >
                            <span className="font-mono text-2xs text-ink-soft">{bar.pages}</span>
                            <span
                                className={cx(
                                    "w-full rounded-sm",
                                    bar.fraction === 1 ? "bg-accent" : "bg-raised-strong",
                                )}
                                style={{ height: `${Math.round(Math.max(bar.fraction, 0.03) * barTrack)}px` }}
                            />
                            <span className="truncate text-2xs text-ink-faint">{copy.reports.depth.level(bar.level)}</span>
                        </li>
                    ))}
                </ul>
            )}
        </Panel>
    );
}

interface LineProps {
    label: string;
    value: number;
    warn?: boolean;
}

function Line({ label, value, warn = false }: LineProps): ReactElement {
    return (
        <div className="flex items-baseline gap-2">
            <span className="min-w-0 flex-1 truncate text-xs text-ink-dim">{label}</span>
            <span className={cx("font-mono text-xs", warn && value > 0 ? "text-danger" : "text-ink")}>{value}</span>
        </div>
    );
}

export interface AuditPanelProps {
    card: AuditCard;
    siteId: string;
}

export function LinkAuditPanel({ card, siteId }: AuditPanelProps): ReactElement {
    return (
        <Panel className="w-80 shrink-0">
            <PanelHeader title={copy.reports.audit.title} />
            <div className="flex flex-col gap-2 p-3">
                <p className="flex items-baseline gap-2">
                    <span className={cx("font-mono text-2xl leading-none", toneClasses[shareTone(card.compliantShare)].ink)}>
                        {shareOf(card.compliantShare)}
                    </span>
                    <span className="min-w-0 flex-1 text-2xs text-ink-dim">{copy.reports.audit.compliant}</span>
                </p>
                <p className="text-2xs text-ink-faint">{copy.reports.audit.audited(card.audited, card.pages)}</p>
                <Line label={copy.reports.audit.missingRequired} value={card.missingRequired} warn={true} />
                <Line label={copy.reports.audit.missing} value={card.missing} />
                <Line label={copy.reports.audit.blocked} value={card.blocked} warn={true} />
                <Line label={copy.reports.audit.offGraph} value={card.offGraph} />
                <Line label={copy.reports.audit.orphans} value={card.orphans} />
                <Link
                    to={`/s/${siteId}/links`}
                    className="flex h-7 items-center justify-center rounded-md border border-hairline text-sm text-ink hover:bg-raised"
                >
                    {copy.reports.audit.open}
                </Link>
            </div>
        </Panel>
    );
}
