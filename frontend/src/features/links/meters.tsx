import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { LinkPolicySummary, LinkTotals } from "../../data/types.js";
import { Button, cx, PlayCircleIcon, SectionLabel, toneClasses } from "../../ui/index.js";
import type { Tone } from "../../ui/index.js";

interface Segment {
    key: string;
    label: string;
    count: number;
    tone: Tone;
    className: string;
}

function segmentsOf(totals: LinkTotals): Segment[] {
    const optional = Math.max(totals.missing - totals.missingRequired, 0);
    return [
        { key: "satisfied", label: copy.links.meters.satisfied, count: totals.satisfied, tone: "ok", className: toneClasses.ok.solid },
        { key: "missingRequired", label: copy.links.meters.missingRequired, count: totals.missingRequired, tone: "danger", className: toneClasses.danger.solid },
        { key: "missingOptional", label: copy.links.meters.missingOptional, count: optional, tone: "warn", className: toneClasses.warn.solid },
        { key: "blocked", label: copy.links.meters.blocked, count: totals.blocked, tone: "danger", className: "bg-danger/45" },
    ];
}

interface TileProps {
    label: string;
    value: string;
    hint: string;
    tone?: Tone;
}

function Tile({ label, value, hint, tone }: TileProps): ReactElement {
    return (
        <div className="flex min-w-32 flex-col gap-0.5 rounded-lg border border-hairline bg-panel px-3 py-2">
            <SectionLabel>{label}</SectionLabel>
            <span className={cx("font-mono text-lg text-ink", tone !== undefined && toneClasses[tone].ink)}>{value}</span>
            <span className="text-2xs text-ink-faint">{hint}</span>
        </div>
    );
}

export interface AuditMetersProps {
    totals: LinkTotals;
    policy: LinkPolicySummary;
    relinkCount: number;
    relinkCapped: boolean;
    relinkCap: number;
    onRelink: () => void;
}

export function AuditMeters({ totals, policy, relinkCount, relinkCapped, relinkCap, onRelink }: AuditMetersProps): ReactElement {
    const segments = segmentsOf(totals);
    const total = Math.max(totals.targets, 1);
    return (
        <div className="flex flex-wrap items-stretch gap-3 border-b border-hairline px-3 py-3">
            <div className="flex min-w-80 flex-1 flex-col gap-2 rounded-lg border border-hairline bg-panel px-3 py-2">
                <div className="flex items-baseline justify-between gap-2">
                    <SectionLabel>{copy.links.meters.required}</SectionLabel>
                    <span className="font-mono text-2xs text-ink-faint">{copy.links.meters.targets(totals.targets)}</span>
                </div>
                <div className="flex h-2 gap-0.5 overflow-hidden rounded-sm bg-inset" role="img" aria-label={copy.links.meters.required}>
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
                <ul className="flex flex-wrap gap-x-4 gap-y-1">
                    {segments.map((segment) => (
                        <li key={segment.key} className="flex items-center gap-1.5 text-2xs text-ink-dim">
                            <span className={cx("h-1.5 w-1.5 rounded-full", segment.className)} aria-hidden={true} />
                            <span className="font-mono text-ink">{segment.count}</span>
                            <span>{segment.label}</span>
                        </li>
                    ))}
                </ul>
                <p className="text-2xs text-ink-faint">
                    {copy.links.policy(policy.name)} · {policy.forbidExternal ? copy.links.forbidExternal : copy.links.allowExternal} ·{" "}
                    {policy.forbidSelf ? copy.links.forbidSelf : copy.links.allowSelf}
                </p>
            </div>
            <Tile label={copy.links.meters.audited} value={copy.links.meters.ofPages(totals.audited, totals.pages)} hint={copy.links.meters.auditedHint} />
            <Tile
                label={copy.links.meters.orphans}
                value={String(totals.orphans)}
                hint={copy.links.meters.orphansHint}
                tone={totals.orphans > 0 ? "warn" : undefined}
            />
            <Tile
                label={copy.links.meters.offGraph}
                value={String(totals.offGraph)}
                hint={copy.links.meters.offGraphHint}
                tone={totals.offGraph > 0 ? "warn" : undefined}
            />
            <div className="flex min-w-44 flex-col justify-between gap-2 rounded-lg border border-hairline bg-panel px-3 py-2">
                <p className="text-2xs text-ink-dim">{copy.links.relink.hint}</p>
                <div className="flex flex-col gap-1">
                    <Button variant="primary" size="sm" icon={PlayCircleIcon} disabled={relinkCount === 0} onClick={onRelink}>
                        {relinkCount === 0 ? copy.links.relink.none : copy.links.relink.start(relinkCount)}
                    </Button>
                    {relinkCapped ? <span className="text-2xs text-warn">{copy.links.relink.capped(relinkCap)}</span> : null}
                </div>
            </div>
        </div>
    );
}
