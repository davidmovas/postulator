import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { cx, ErrorIcon, InfoIcon, WarningIcon, toneClasses } from "../../ui/index.js";
import type { IconComponent, Tone } from "../../ui/index.js";
import type { Finding, Severity } from "./artifacts.js";
import { severityError, severityWarn, weigh } from "./artifacts.js";

const severityTones: Readonly<Record<Severity, Tone>> = {
    error: "danger",
    warn: "warn",
    info: "info",
};

const severityIcons: Readonly<Record<Severity, IconComponent>> = {
    error: ErrorIcon,
    warn: WarningIcon,
    info: InfoIcon,
};

export interface FindingListProps {
    findings: readonly Finding[];
    empty: string;
}

export function FindingList({ findings, empty }: FindingListProps): ReactElement {
    if (findings.length === 0) {
        return <p className="px-3 py-2 text-xs text-ink-dim">{empty}</p>;
    }
    const ranked = [...findings].sort((left, right) => rank(right.severity) - rank(left.severity));
    return (
        <ul className="flex flex-col">
            {ranked.map((finding, position) => {
                const Icon = severityIcons[finding.severity];
                const tone = toneClasses[severityTones[finding.severity]];
                return (
                    <li
                        key={`${finding.code}:${String(position)}`}
                        className="flex gap-2 border-b border-inset px-3 py-2 last:border-b-0"
                    >
                        <Icon size={14} className={cx("mt-0.5 shrink-0", tone.ink)} />
                        <div className="flex min-w-0 flex-col gap-0.5">
                            <span className="font-mono text-2xs text-ink-faint">{finding.code}</span>
                            <span className="text-xs text-ink-soft">{finding.message}</span>
                        </div>
                    </li>
                );
            })}
        </ul>
    );
}

function rank(severity: Severity): number {
    if (severity === severityError) {
        return 2;
    }
    return severity === severityWarn ? 1 : 0;
}

export interface FindingTotalsProps {
    findings: readonly Finding[];
}

export function FindingTotals({ findings }: FindingTotalsProps): ReactElement | null {
    const counted = weigh(findings);
    if (counted.errors === 0 && counted.warnings === 0) {
        return null;
    }
    return (
        <div className="flex flex-wrap items-center gap-x-3 px-3 py-1.5 text-2xs">
            {counted.errors === 0 ? null : (
                <span className="text-danger">{copy.runs.review.links.errors(counted.errors)}</span>
            )}
            {counted.warnings === 0 ? null : (
                <span className="text-warn">{copy.runs.review.links.warnings(counted.warnings)}</span>
            )}
            <span className="text-ink-faint">{copy.runs.review.links.warningsDoNotFail}</span>
        </div>
    );
}
