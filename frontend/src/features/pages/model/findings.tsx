import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { ModelFinding } from "../../../data/types.js";
import { absoluteTime, relativeTime } from "../../../domain/format.js";
import { CountBadge, Panel, PanelHeader, StatusBadge } from "../../../ui/index.js";
import { Note } from "./facts.js";
import { findingCodeLabel, severityLabel, severityTone } from "./labels.js";
import { findingSentence } from "../../findings/model/subject.js";

const said = copy.pages.model.findings;

export interface FindingsPaneProps {
    findings: readonly ModelFinding[];
}

export function FindingsPane({ findings }: FindingsPaneProps): ReactElement {
    return (
        <Panel>
            <PanelHeader title={said.title}>
                {findings.length === 0 ? null : <CountBadge tone="warn" count={findings.length} />}
            </PanelHeader>
            {findings.length === 0 ? (
                <Note>{said.none}</Note>
            ) : (
                <ul className="flex flex-col">
                    {findings.map((held) => (
                        <li key={held.id} className="flex flex-col gap-1 border-b border-inset px-3 py-2 last:border-b-0">
                            <div className="flex min-w-0 items-center gap-2">
                                <StatusBadge tone={severityTone(held.severity)}>
                                    <span title={severityLabel(held.severity)}>{findingCodeLabel(held.code)}</span>
                                </StatusBadge>
                                <span
                                    className="ml-auto shrink-0 font-mono text-2xs text-ink-faint"
                                    title={absoluteTime(held.createdAt)}
                                >
                                    {relativeTime(held.createdAt)}
                                </span>
                            </div>
                            <p className="text-xs text-ink-soft">{findingSentence(held.code, held.subject)}</p>
                        </li>
                    ))}
                </ul>
            )}
        </Panel>
    );
}
