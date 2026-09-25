import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { useArtifact } from "../../../data/hooks/runs.js";
import { cx, SectionLabel, StatusBadge, toneClasses } from "../../../ui/index.js";
import type { ActualLink, Finding, LinkClass, LinkContextView, LinkTargetView, OwedTarget } from "../artifacts.js";
import {
    actualLinks,
    countByClass,
    decodeArtifact,
    linkClasses,
    linkContextView,
    owedTargets,
    validationView,
} from "../artifacts.js";
import { FindingList, FindingTotals } from "../findings.js";
import { linkClassLabel, linkClassTone } from "../labels.js";
import { artifactLinkContext } from "../statuses.js";
import { Unreadable } from "./shared.js";

function relationLabel(relation: string): string {
    const table = copy.runs.review.links.relation;
    if (relation === "up" || relation === "down" || relation === "sibling") {
        return table[relation];
    }
    return relation;
}

function TargetRows({ targets }: { targets: readonly LinkTargetView[] }): ReactElement {
    return (
        <ul className="flex flex-col">
            {targets.map((target) => (
                <li
                    key={`${target.relation}:${target.pageId}:${target.url}`}
                    className="flex items-center gap-2 border-b border-inset px-3 py-1.5 text-xs last:border-b-0"
                >
                    <span className="w-16 shrink-0 text-2xs text-ink-faint">{relationLabel(target.relation)}</span>
                    <span className="min-w-0 flex-1 truncate font-mono text-ink-soft">{target.url}</span>
                    <span className="shrink-0 text-2xs text-ink-faint">
                        {target.required ? copy.runs.review.links.required : copy.runs.review.links.optional}
                    </span>
                    <span className="w-40 shrink-0 truncate text-2xs text-ink-faint">
                        {target.anchors.join(", ")}
                    </span>
                </li>
            ))}
        </ul>
    );
}

export interface LinkContextPaneProps {
    context: LinkContextView | null;
}

export function LinkContextPane({ context }: LinkContextPaneProps): ReactElement {
    if (context === null) {
        return <Unreadable />;
    }
    if (context.targets.length === 0) {
        return <p className="px-3 py-2 text-xs text-ink-dim">{copy.runs.review.links.owedEmpty}</p>;
    }
    return (
        <div className="flex flex-col">
            <SectionLabel className="px-3 pt-2">{copy.runs.review.links.owed}</SectionLabel>
            <TargetRows targets={context.targets} />
        </div>
    );
}

function OwedRows({ owed }: { owed: readonly OwedTarget[] }): ReactElement {
    return (
        <ul className="flex flex-col">
            {owed.map((entry) => (
                <li
                    key={`${entry.target.relation}:${entry.target.pageId}:${entry.target.url}`}
                    className="flex items-center gap-2 border-b border-inset px-3 py-1.5 text-xs last:border-b-0"
                >
                    <span className="w-16 shrink-0 text-2xs text-ink-faint">
                        {relationLabel(entry.target.relation)}
                    </span>
                    <span className="min-w-0 flex-1 truncate font-mono text-ink-soft">{entry.target.url}</span>
                    <span className="w-40 shrink-0 truncate text-2xs text-ink-faint">{entry.anchor}</span>
                    <StatusBadge tone={entry.satisfied ? "ok" : entry.target.required ? "danger" : "warn"}>
                        {entry.satisfied ? copy.runs.review.links.satisfied : copy.runs.review.links.missing}
                    </StatusBadge>
                </li>
            ))}
        </ul>
    );
}

function ActualRows({ links }: { links: readonly ActualLink[] }): ReactElement {
    const counts = countByClass(links);
    return (
        <div className="flex flex-col">
            <div className="flex flex-wrap items-center gap-2 px-3 py-1.5" title={copy.runs.review.links.classesNote}>
                {linkClasses.map((kind: LinkClass) => (
                    <StatusBadge key={kind} tone={linkClassTone(kind)} dot={false}>
                        {`${linkClassLabel(kind)} ${String(counts[kind])}`}
                    </StatusBadge>
                ))}
            </div>
            {links.length === 0 ? (
                <p className="px-3 py-2 text-xs text-ink-dim">{copy.runs.review.links.actualEmpty}</p>
            ) : (
                <ul className="flex flex-col">
                    {links.map((link, position) => (
                        <li
                            key={`${link.kind}:${link.href}:${String(position)}`}
                            className="flex items-center gap-2 border-b border-inset px-3 py-1.5 text-xs last:border-b-0"
                        >
                            <span className="min-w-0 flex-1 truncate font-mono text-ink-soft">{link.href}</span>
                            <span className="w-40 shrink-0 truncate text-2xs text-ink-faint">{link.anchor}</span>
                            <span className={cx("shrink-0 text-2xs", toneClasses[linkClassTone(link.kind)].ink)}>
                                {linkClassLabel(link.kind)}
                            </span>
                        </li>
                    ))}
                </ul>
            )}
        </div>
    );
}

export interface ValidationPaneProps {
    itemId: string;
    payload: unknown;
}

export function ValidationPane({ itemId, payload }: ValidationPaneProps): ReactElement {
    const view = validationView(payload);
    const context = useArtifact(itemId, artifactLinkContext);
    const declared = linkContextView(
        context.data === undefined ? null : decodeArtifact(context.data.artifact.content),
    );

    if (view === null) {
        return <Unreadable />;
    }

    const findings: readonly Finding[] = [...view.compliance, ...view.structure];
    const owed = owedTargets(declared, view);

    return (
        <div className="flex flex-col gap-2 pb-3">
            <div className="flex items-center justify-between gap-2 px-3 pt-2">
                <SectionLabel>{copy.runs.review.links.owed}</SectionLabel>
                {view.score === null ? null : (
                    <span className="font-mono text-2xs text-ink-faint">
                        {copy.runs.review.links.score(view.score)}
                    </span>
                )}
            </div>
            <p className="px-3 text-2xs text-ink-faint">{copy.runs.review.links.snapshot}</p>
            {owed.length === 0 ? (
                <p className="px-3 text-xs text-ink-dim">{copy.runs.review.links.owedEmpty}</p>
            ) : (
                <OwedRows owed={owed} />
            )}

            <SectionLabel className="px-3 pt-2">{copy.runs.review.links.actual}</SectionLabel>
            <ActualRows links={actualLinks(view)} />

            <SectionLabel className="px-3 pt-2">{copy.runs.review.links.findings}</SectionLabel>
            <FindingTotals findings={findings} />
            <FindingList findings={findings} empty={copy.runs.review.links.clean} />
        </div>
    );
}
