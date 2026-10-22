import type { ReactElement, ReactNode } from "react";

import { copy } from "../../../copy/index.js";
import { absoluteTime } from "../../../domain/format.js";
import { SectionLabel } from "../../../ui/index.js";
import { fieldLabel } from "../../settings/types/labels.js";
import type { RevertedFieldView } from "../artifacts.js";
import { finalView, publishView, relinkView, revertView, syncView } from "../artifacts.js";
import { revertedFieldLabel } from "../diff-labels.js";
import { FindingList, FindingTotals } from "../findings.js";
import { revertOutcomeLabel } from "../labels.js";
import { SiteRegions } from "./regions.js";
import { ExternalUrl, Rows, Unreadable } from "./shared.js";
import { PublishDiff } from "./written.js";

export interface PayloadPaneProps {
    payload: unknown;
}

export interface PublishPaneProps extends PayloadPaneProps {
    pageId: string;
    runId: string;
}

function LiveUrl({ url }: { url: string }): ReactElement | null {
    if (url === "") {
        return null;
    }
    return (
        <div className="flex min-w-0 flex-col gap-0.5 px-3">
            <SectionLabel>{copy.runs.review.publish.liveUrl}</SectionLabel>
            <ExternalUrl url={url} />
        </div>
    );
}

export function PublishPane({ payload, pageId, runId }: PublishPaneProps): ReactElement {
    const view = publishView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    const entries: (readonly [string, ReactNode])[] = [
        [
            copy.runs.review.publish.result,
            view.created ? copy.runs.review.publish.created : copy.runs.review.publish.updated,
        ],
        [copy.runs.review.publish.status, view.status],
        [copy.runs.review.publish.wpId, view.wpId === null ? "" : String(view.wpId)],
        [copy.runs.review.publish.hash, view.contentHash],
        [copy.runs.review.publish.seoApplied, view.seoApplied.join(", ")],
    ];
    if (view.skipped.length > 0) {
        entries.push([copy.runs.review.publish.skipped, view.skipped.join(", ")]);
    }
    if (view.product !== null) {
        const said = copy.runs.review.publish;
        entries.push(
            [said.productShort, view.product.shortWritten ? said.productShortWritten : said.productShortKept],
            [said.productAdded, view.product.added.length === 0 ? said.productAddedNone : view.product.added.join(", ")],
            [said.productImage, view.product.imageSet ? said.productImageSet : said.productImageKept],
        );
    }
    return (
        <div className="flex flex-col gap-2 pb-3" title={copy.runs.review.noDiff}>
            <Rows entries={entries} />
            <LiveUrl url={view.url} />
            {view.recorded ? <PublishDiff view={view} pageId={pageId} runId={runId} /> : null}
            {pageId === "" ? null : <SiteRegions pageId={pageId} />}
            <FindingTotals findings={view.findings} />
            <FindingList findings={view.findings} empty={copy.runs.review.links.clean} />
        </div>
    );
}

function RevertedFields({ fields }: { fields: readonly RevertedFieldView[] }): ReactElement | null {
    if (fields.length === 0) {
        return null;
    }
    const said = copy.runs.review.revert;
    return (
        <section aria-label={said.fields} className="flex flex-col">
            <SectionLabel className="px-3">{said.fields}</SectionLabel>
            <ul className="flex flex-col">
                {fields.map((field) => (
                    <li key={field.key} className="flex flex-col gap-0.5 border-b border-inset px-3 py-1.5 text-xs last:border-b-0">
                        <span className="flex items-center gap-2">
                            <span className="min-w-0 flex-1 truncate text-ink">{fieldLabel(field.key)}</span>
                            <span className="shrink-0 text-2xs text-ink-faint">{revertedFieldLabel(field.outcome)}</span>
                        </span>
                        {field.detail === "" ? null : <p className="text-2xs text-ink-dim">{field.detail}</p>}
                    </li>
                ))}
            </ul>
        </section>
    );
}

export function RevertPane({ payload }: PayloadPaneProps): ReactElement {
    const view = revertView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    const said = copy.runs.review.revert;
    return (
        <div className="flex flex-col gap-2 pb-3">
            <Rows
                entries={[
                    [copy.pages.detail.path, view.path],
                    [said.outcome, revertOutcomeLabel(view.outcome)],
                    [said.detail, view.detail],
                ]}
            />
            <RevertedFields fields={view.fields} />
            <SectionLabel className="px-3">{said.neighbours}</SectionLabel>
            {view.neighbours.length === 0 ? (
                <p className="px-3 text-xs text-ink-dim">{said.none}</p>
            ) : (
                <ul className="flex flex-col">
                    {view.neighbours.map((neighbour) => (
                        <li
                            key={neighbour.pageId}
                            className="flex items-center gap-2 border-b border-inset px-3 py-1.5 text-xs last:border-b-0"
                        >
                            <span className="min-w-0 flex-1 truncate font-mono text-ink-soft">{neighbour.path}</span>
                            <span className="w-40 shrink-0 truncate text-2xs text-ink-faint" title={neighbour.detail}>
                                {revertOutcomeLabel(neighbour.outcome)}
                            </span>
                        </li>
                    ))}
                </ul>
            )}
            <FindingTotals findings={view.findings} />
            <FindingList findings={view.findings} empty={copy.runs.review.links.clean} />
        </div>
    );
}

export function RelinkPane({ payload }: PayloadPaneProps): ReactElement {
    const view = relinkView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    return (
        <div className="flex flex-col gap-2 pb-3">
            <Rows
                entries={[
                    [copy.runs.review.relink.linked, view.linked === null ? "" : String(view.linked)],
                    [copy.runs.review.relink.conflicts, view.conflicts === null ? "" : String(view.conflicts)],
                    [copy.runs.review.relink.skipped, view.skipped === null ? "" : String(view.skipped)],
                ]}
            />
            <SectionLabel className="px-3">{copy.runs.review.relink.neighbours}</SectionLabel>
            {view.neighbours.length === 0 ? (
                <p className="px-3 text-xs text-ink-dim">{copy.runs.review.relink.none}</p>
            ) : (
                <ul className="flex flex-col">
                    {view.neighbours.map((neighbour) => (
                        <li
                            key={neighbour.pageId}
                            className="flex items-center gap-2 border-b border-inset px-3 py-1.5 text-xs last:border-b-0"
                        >
                            <span className="min-w-0 flex-1 truncate font-mono text-ink-soft">{neighbour.path}</span>
                            <span className="w-40 shrink-0 truncate text-2xs text-ink-faint">{neighbour.anchor}</span>
                            <span className="w-28 shrink-0 truncate text-2xs text-ink-faint">{neighbour.outcome}</span>
                        </li>
                    ))}
                </ul>
            )}
            <FindingList findings={view.findings} empty={copy.runs.review.links.clean} />
        </div>
    );
}

export function SyncPane({ payload }: PayloadPaneProps): ReactElement {
    const view = syncView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    return (
        <div className="flex flex-col gap-2 pb-3">
            <Rows
                entries={[
                    [copy.runs.review.publish.status, view.status],
                    [copy.runs.review.publish.wpId, view.wpId === null ? "" : String(view.wpId)],
                    [copy.runs.review.publish.hash, view.contentHash],
                    [copy.pages.detail.links, view.links === null ? "" : String(view.links)],
                    [copy.pages.drift.wpModified, absoluteTime(view.modifiedAt)],
                ]}
            />
            <LiveUrl url={view.url} />
            {view.findings.length === 0 ? null : (
                <FindingList findings={view.findings} empty={copy.runs.review.links.clean} />
            )}
        </div>
    );
}

export function FinalPane({ payload }: PayloadPaneProps): ReactElement {
    const view = finalView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    return (
        <div className="flex flex-col gap-2 pb-3">
            <Rows
                entries={[
                    [copy.runs.review.report.score, view.score === null ? "" : view.score.toFixed(2)],
                    [copy.runs.review.report.errors, view.errors === null ? "" : String(view.errors)],
                    [copy.runs.review.report.warnings, view.warnings === null ? "" : String(view.warnings)],
                    [copy.pages.detail.path, view.path],
                ]}
            />
            <FindingTotals findings={view.findings} />
            <FindingList findings={view.findings} empty={copy.runs.review.links.clean} />
        </div>
    );
}
