import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { LinkPolicySummary, LinkTotals } from "../../data/types.js";
import { cx, toneClasses } from "../../ui/index.js";
import type { Tone } from "../../ui/index.js";

interface Segment {
    key: string;
    label: string;
    count: number;
    className: string;
}

function segmentsOf(totals: LinkTotals): Segment[] {
    const optional = Math.max(totals.missing - totals.missingRequired, 0);
    return [
        { key: "satisfied", label: copy.links.meters.satisfied, count: totals.satisfied, className: toneClasses.ok.solid },
        { key: "missingRequired", label: copy.links.meters.missingRequired, count: totals.missingRequired, className: toneClasses.danger.solid },
        { key: "missingOptional", label: copy.links.meters.missingOptional, count: optional, className: toneClasses.warn.solid },
        { key: "blocked", label: copy.links.meters.blocked, count: totals.blocked, className: "bg-danger/45" },
    ];
}

interface StatProps {
    label: string;
    value: string;
    tone?: Tone;
}

function Stat({ label, value, tone }: StatProps): ReactElement {
    return (
        <span className="flex items-baseline gap-1.5 whitespace-nowrap text-2xs text-ink-dim">
            <span className={cx("font-mono text-xs text-ink", tone !== undefined && toneClasses[tone].ink)}>{value}</span>
            <span>{label}</span>
        </span>
    );
}

function policyLine(policy: LinkPolicySummary): string {
    return [
        copy.links.policy(policy.name),
        policy.forbidExternal ? copy.links.forbidExternal : copy.links.allowExternal,
        policy.forbidSelf ? copy.links.forbidSelf : copy.links.allowSelf,
    ].join(" · ");
}

export interface AuditStripProps {
    totals: LinkTotals;
    policy: LinkPolicySummary;
}

export function AuditStrip({ totals, policy }: AuditStripProps): ReactElement {
    const segments = segmentsOf(totals);
    const total = Math.max(totals.targets, 1);
    return (
        <div className="flex shrink-0 items-center gap-4 border-b border-hairline px-4 py-2">
            <div className="flex min-w-0 flex-1 flex-col gap-1.5" title={policyLine(policy)}>
                <div className="flex h-1.5 gap-0.5 overflow-hidden rounded-sm bg-inset" role="img" aria-label={copy.links.meters.required}>
                    {segments
                        .filter((segment) => segment.count > 0)
                        .map((segment) => (
                            <span
                                key={segment.key}
                                className={cx("block h-full rounded-sm", segment.className)}
                                style={{ width: `${(segment.count / total) * 100}%` }}
                            />
                        ))}
                </div>
                <ul className="flex flex-wrap gap-x-3 gap-y-0.5">
                    {segments.map((segment) => (
                        <li key={segment.key} className="flex items-center gap-1.5 text-2xs text-ink-dim">
                            <span className={cx("h-1.5 w-1.5 rounded-full", segment.className)} aria-hidden={true} />
                            <span className="font-mono text-ink">{segment.count}</span>
                            <span>{segment.label}</span>
                        </li>
                    ))}
                    <li className="text-2xs text-ink-faint">{copy.links.meters.targets(totals.targets)}</li>
                </ul>
            </div>
            <div className="flex shrink-0 items-center gap-4">
                <Stat label={copy.links.meters.audited} value={copy.links.meters.ofPages(totals.audited, totals.pages)} />
                <Stat label={copy.links.meters.orphans} value={String(totals.orphans)} tone={totals.orphans > 0 ? "warn" : undefined} />
                <Stat label={copy.links.meters.offGraph} value={String(totals.offGraph)} tone={totals.offGraph > 0 ? "warn" : undefined} />
            </div>
        </div>
    );
}
