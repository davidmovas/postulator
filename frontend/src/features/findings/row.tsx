import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { Finding, FindingChoice } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import type { IconComponent } from "../../ui/index.js";
import { Button, EditNoteIcon, HubIcon, PlayArrowIcon, StatusBadge } from "../../ui/index.js";
import { findingCodeLabel, severityLabel, severityTone } from "../pages/model/labels.js";
import { optionDoes, optionLabel, resolutionLabel, resolvedByLabel } from "./labels.js";
import { findingSentence } from "./model/subject.js";

const said = copy.findings;

export const findingRowHeight = 88;

const openStatus = "open";
const runKind = "run";
const proposalKind = "proposal";
const cardCode = "entity_card_proposed";
const acceptOption = "accept";

export function onTheGraph(finding: Finding, option: string): boolean {
    return finding.code === cardCode && option === acceptOption;
}

function iconOf(option: FindingChoice, graph: boolean): IconComponent | undefined {
    if (option.kind === runKind) {
        return PlayArrowIcon;
    }
    if (option.kind === proposalKind) {
        return HubIcon;
    }
    return graph ? EditNoteIcon : undefined;
}

function OptionButton({ finding, option, onChoose }: { finding: Finding; option: FindingChoice; onChoose: () => void }): ReactElement {
    const graph = onTheGraph(finding, option.name);
    return (
        <Button
            size="sm"
            variant="secondary"
            icon={iconOf(option, graph)}
            title={graph ? said.cards : optionDoes(option.name)}
            data-finding-option={option.name}
            onClick={onChoose}
        >
            {optionLabel(option.name)}
        </Button>
    );
}

export interface FindingRowProps {
    finding: Finding;
    onOpenPage: (pageId: string) => void;
    onChoose: (finding: Finding, option: FindingChoice) => void;
}

export function FindingRow({ finding, onOpenPage, onChoose }: FindingRowProps): ReactElement {
    const sentence = findingSentence(finding.code, finding.subject);
    const pageId = finding.pageId ?? "";
    const open = finding.status === openStatus;
    return (
        <article
            aria-label={findingCodeLabel(finding.code)}
            data-finding={finding.id}
            style={{ height: findingRowHeight }}
            className="flex flex-col justify-center gap-1 border-b border-inset px-3"
        >
            <div className="flex min-w-0 items-center gap-2">
                <StatusBadge tone={severityTone(finding.severity)}>
                    <span title={severityLabel(finding.severity)}>{findingCodeLabel(finding.code)}</span>
                </StatusBadge>
                {pageId === "" ? (
                    <span className="truncate text-xs text-ink-dim">{said.site}</span>
                ) : (
                    <button
                        type="button"
                        title={said.openPage}
                        onClick={() => {
                            onOpenPage(pageId);
                        }}
                        className="flex min-w-0 items-center gap-2 rounded-sm text-xs text-ink hover:text-accent"
                    >
                        <span className="truncate font-medium">{finding.pageTitle ?? ""}</span>
                        <span className="truncate font-mono text-2xs text-ink-faint">{finding.pagePath ?? ""}</span>
                    </button>
                )}
                <span className="ml-auto shrink-0 font-mono text-2xs text-ink-faint" title={absoluteTime(finding.createdAt)}>
                    {relativeTime(finding.createdAt)}
                </span>
            </div>
            <p className="truncate text-xs text-ink-soft" title={sentence}>
                {sentence}
            </p>
            <div className="flex min-w-0 items-center gap-1.5">
                {open ? (
                    (finding.options ?? []).map((option) => (
                        <OptionButton
                            key={option.name}
                            finding={finding}
                            option={option}
                            onChoose={() => {
                                onChoose(finding, option);
                            }}
                        />
                    ))
                ) : (
                    <span className="truncate text-2xs text-ink-faint" title={absoluteTime(finding.updatedAt)}>
                        {said.settled(
                            resolvedByLabel(finding.resolvedBy ?? ""),
                            relativeTime(finding.updatedAt),
                            resolutionLabel(finding.resolution ?? ""),
                        )}
                    </span>
                )}
            </div>
        </article>
    );
}
