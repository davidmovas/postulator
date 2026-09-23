import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { ImportConflict, ImportFinding } from "../../data/types.js";
import { EmptyState, Panel, PanelHeader, StatusBadge, TaskAltIcon } from "../../ui/index.js";
import { findingLabel, reasonLabel } from "./labels.js";

interface GroupProps {
    title: string;
    count: number;
    tone: "danger" | "warn";
    findings: readonly ImportFinding[];
}

function Group({ title, count, tone, findings }: GroupProps): ReactElement | null {
    if (count === 0) {
        return null;
    }
    return (
        <Panel>
            <PanelHeader title={title}>
                <StatusBadge tone={tone} dot={false}>
                    {String(count)}
                </StatusBadge>
            </PanelHeader>
            <ul className="flex flex-col">
                {findings.map((finding, at) => (
                    <li
                        key={`${finding.row}-${finding.code}-${at}`}
                        className="flex flex-col gap-0.5 border-b border-hairline px-3 py-2 last:border-b-0"
                    >
                        <div className="flex items-baseline gap-2">
                            {finding.row > 0 ? (
                                <span className="shrink-0 font-mono text-2xs text-ink-faint">
                                    {copy.imports.preview.row(finding.row)}
                                </span>
                            ) : null}
                            <span className="min-w-0 flex-1 text-xs text-ink">{findingLabel(finding.code)}</span>
                        </div>
                        {finding.message === "" ? null : (
                            <p className="text-2xs text-ink-dim">{finding.message}</p>
                        )}
                    </li>
                ))}
            </ul>
        </Panel>
    );
}

export interface FindingsPanelProps {
    errors: readonly ImportFinding[];
    warnings: readonly ImportFinding[];
    conflicts: readonly ImportConflict[];
}

export function FindingsPanel({ errors, warnings, conflicts }: FindingsPanelProps): ReactElement {
    const quiet = errors.length === 0 && warnings.length === 0 && conflicts.length === 0;
    return (
        <div className="flex flex-col gap-3 p-3">
            <Group title={copy.imports.preview.errors} count={errors.length} tone="danger" findings={errors} />
            <Group title={copy.imports.preview.warnings} count={warnings.length} tone="warn" findings={warnings} />
            {conflicts.length === 0 ? null : (
                <Panel>
                    <PanelHeader title={copy.imports.preview.conflicts}>
                        <StatusBadge tone="warn" dot={false}>
                            {String(conflicts.length)}
                        </StatusBadge>
                    </PanelHeader>
                    <ul className="flex flex-col">
                        {conflicts.map((conflict, at) => (
                            <li
                                key={`${conflict.path}-${at}`}
                                className="flex flex-col gap-0.5 border-b border-hairline px-3 py-2 last:border-b-0"
                            >
                                <span className="truncate font-mono text-xs text-ink" title={conflict.path}>
                                    {conflict.path}
                                </span>
                                <span className="text-2xs text-ink-dim">{reasonLabel(conflict.reason)}</span>
                            </li>
                        ))}
                    </ul>
                </Panel>
            )}
            {quiet ? <EmptyState icon={TaskAltIcon} title={copy.empty.importFindings} /> : null}
        </div>
    );
}
