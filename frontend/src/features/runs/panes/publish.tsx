import type { ReactElement, ReactNode } from "react";

import { copy } from "../../../copy/index.js";
import { filedItems } from "../../../domain/categories.js";
import { absoluteTime } from "../../../domain/format.js";
import { CategoryTrail, SectionLabel } from "../../../ui/index.js";
import type { CategoryWriteView } from "../artifacts.js";
import { finalView, publishView, relinkView, revertView, syncView } from "../artifacts.js";
import { FindingList, FindingTotals } from "../findings.js";
import { revertOutcomeLabel } from "../labels.js";
import { ExternalUrl, Rows, Unreadable } from "./shared.js";

export interface PayloadPaneProps {
    payload: unknown;
}

const productTaxonomy = "product_cat";

function termNames(ids: readonly number[], write: CategoryWriteView): string {
    return ids
        .map((termId) => {
            const named = write.terms.find((term) => term.termId === termId);
            return named === undefined || named.name === "" ? `#${String(termId)}` : named.name;
        })
        .join(", ");
}

function FiledUnder({ write }: { write: CategoryWriteView }): ReactElement | null {
    if (write.terms.length === 0) {
        return null;
    }
    const said = copy.runs.review.publish;
    const created = write.terms.filter((term) => term.created).map((term) => term.name);
    return (
        <div data-publish-categories={true} className="flex flex-col">
            <div className="flex flex-col gap-1 px-3 pt-1">
                <SectionLabel>{write.taxonomy === productTaxonomy ? said.filedUnderProducts : said.filedUnder}</SectionLabel>
                <CategoryTrail items={filedItems(write.terms)} label={copy.categories.trail} />
            </div>
            <Rows
                entries={[
                    [said.termsCreated, created.length === 0 ? said.termsCreatedNone : created.join(", ")],
                    [said.termsAdded, write.added.length === 0 ? said.termsAddedNone : termNames(write.added, write)],
                    [said.termsBefore, write.previous.length === 0 ? said.termsBeforeNone : termNames(write.previous, write)],
                    [said.termsTaken, write.taken ? said.termsTakenYes : said.termsTakenNo],
                ]}
            />
        </div>
    );
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

export function PublishPane({ payload }: PayloadPaneProps): ReactElement {
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
            {view.categories === null ? null : <FiledUnder write={view.categories} />}
            <FindingTotals findings={view.findings} />
            <FindingList findings={view.findings} empty={copy.runs.review.links.clean} />
        </div>
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
