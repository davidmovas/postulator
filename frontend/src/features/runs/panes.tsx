import type { ReactElement, ReactNode } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { failure } from "../../data/errors.js";
import { useJudgePage } from "../../data/hooks/reports.js";
import { useArtifact } from "../../data/hooks/runs.js";
import { isBrowsable, openExternal } from "../../data/host.js";
import { absoluteTime, bytes, relativeTime } from "../../domain/format.js";
import type { ArtifactKind } from "../../generated/vocab.js";
import {
    Banner,
    Button,
    cx,
    EmptyState,
    GavelIcon,
    HtmlPreview,
    OpenInNewIcon,
    SectionLabel,
    SkeletonRows,
    StatusBadge,
    TaskAltIcon,
    toneClasses,
} from "../../ui/index.js";
import type { ActualLink, Finding, LinkClass, LinkTargetView, OwedTarget } from "./artifacts.js";
import {
    actualLinks,
    countByClass,
    decodeArtifact,
    draftView,
    finalView,
    imagesView,
    judgeView,
    linkClasses,
    linkContextView,
    metaView,
    owedTargets,
    publishView,
    relinkView,
    syncView,
    validationView,
} from "./artifacts.js";
import { FindingList, FindingTotals } from "./findings.js";
import { linkClassLabel, linkClassTone, retentionIcon } from "./labels.js";
import {
    artifactBodyHtml,
    artifactDraft,
    artifactFinalReport,
    artifactImages,
    artifactJudgeReport,
    artifactLinkContext,
    artifactMeta,
    artifactPublishResult,
    artifactRelinkResult,
    artifactSyncResult,
    artifactValidationReport,
} from "./statuses.js";

interface RowsProps {
    entries: readonly (readonly [string, ReactNode])[];
}

function Rows({ entries }: RowsProps): ReactElement {
    return (
        <dl className="grid grid-cols-[minmax(0,10rem)_1fr] gap-x-3 gap-y-1 px-3 py-2 text-xs">
            {entries.map(([label, value]) => (
                <div key={label} className="contents">
                    <dt className="truncate text-ink-faint">{label}</dt>
                    <dd className="min-w-0 truncate font-mono text-ink-soft">{value}</dd>
                </div>
            ))}
        </dl>
    );
}

interface ExternalUrlProps {
    url: string;
}

function ExternalUrl({ url }: ExternalUrlProps): ReactElement {
    if (!isBrowsable(url)) {
        return <span className="truncate font-mono text-xs text-ink-soft select-all">{url}</span>;
    }
    return (
        <button
            type="button"
            title={copy.app.openExternal}
            onClick={() => {
                void openExternal(url);
            }}
            className="flex min-w-0 items-center gap-1 text-left font-mono text-xs text-accent hover:underline"
        >
            <span className="truncate">{url}</span>
            <OpenInNewIcon size={12} className="shrink-0" />
        </button>
    );
}

function Unreadable(): ReactElement {
    return (
        <div className="p-3">
            <Banner tone="warn" title={copy.runs.review.unreadable} body={copy.runs.review.unreadableBody} />
        </div>
    );
}

function relationLabel(relation: string): string {
    const table = copy.runs.review.links.relation;
    if (relation === "up" || relation === "down" || relation === "sibling") {
        return table[relation];
    }
    return relation;
}

interface LinkContextPaneProps {
    context: ReturnType<typeof linkContextView>;
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

function LinkContextPane({ context }: LinkContextPaneProps): ReactElement {
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

interface OwedRowsProps {
    owed: readonly OwedTarget[];
}

function OwedRows({ owed }: OwedRowsProps): ReactElement {
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

interface ActualRowsProps {
    links: readonly ActualLink[];
}

function ActualRows({ links }: ActualRowsProps): ReactElement {
    const counts = countByClass(links);
    return (
        <div className="flex flex-col">
            <div className="flex flex-wrap items-center gap-2 px-3 py-1.5">
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
            <p className="px-3 pb-2 text-2xs text-ink-faint">{copy.runs.review.links.classesNote}</p>
        </div>
    );
}

interface ValidationPaneProps {
    itemId: string;
    payload: unknown;
}

function ValidationPane({ itemId, payload }: ValidationPaneProps): ReactElement {
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

interface JudgePaneProps {
    payload: unknown;
    pageId: string;
}

function JudgePane({ payload, pageId }: JudgePaneProps): ReactElement {
    const stored = judgeView(payload);
    const judge = useJudgePage();
    const [fresh, setFresh] = useState<ReturnType<typeof judgeView>>(null);
    const view = fresh ?? stored;

    return (
        <div className="flex flex-col gap-3 p-3">
            {view === null ? (
                <p className="text-xs text-ink-dim">{copy.runs.review.judge.unavailable}</p>
            ) : (
                <>
                    <div className="flex items-end gap-2">
                        <span className="font-mono text-3xl leading-none text-ink">{view.score.toFixed(2)}</span>
                        <span className="pb-0.5 text-2xs text-ink-faint">{copy.runs.review.judge.outOf}</span>
                    </div>
                    <div>
                        <SectionLabel>{copy.runs.review.judge.issues}</SectionLabel>
                        {view.issues.length === 0 ? (
                            <p className="pt-1 text-xs text-ink-dim">{copy.runs.review.judge.none}</p>
                        ) : (
                            <ul className="flex flex-col gap-1 pt-1">
                                {view.issues.map((issue) => (
                                    <li key={issue} className="text-xs text-ink-soft">
                                        {issue}
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                    <div>
                        <SectionLabel>{copy.runs.review.judge.suggestions}</SectionLabel>
                        {view.suggestions.length === 0 ? (
                            <p className="pt-1 text-xs text-ink-dim">{copy.runs.review.judge.none}</p>
                        ) : (
                            <ul className="flex flex-col gap-1 pt-1">
                                {view.suggestions.map((suggestion) => (
                                    <li key={suggestion} className="text-xs text-ink-soft">
                                        {suggestion}
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                </>
            )}
            <div className="flex flex-col gap-1 border-t border-hairline pt-3">
                <Button
                    size="sm"
                    icon={GavelIcon}
                    busy={judge.isPending}
                    disabled={pageId === ""}
                    onClick={() => {
                        judge.mutate(
                            { pageId },
                            {
                                onSuccess: (answered) => {
                                    setFresh(judgeView(answered.report));
                                },
                            },
                        );
                    }}
                >
                    {judge.isPending ? copy.runs.review.judge.rejudging : copy.runs.review.judge.rejudge}
                </Button>
                <p className="text-2xs text-ink-faint">{copy.runs.review.judge.rejudgeHint}</p>
                {judge.data === undefined ? null : (
                    <p className="text-2xs text-ok">{copy.runs.review.judge.rejudged(judge.data.tokens)}</p>
                )}
            </div>
        </div>
    );
}

interface PublishPaneProps {
    payload: unknown;
}

function PublishPane({ payload }: PublishPaneProps): ReactElement {
    const view = publishView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    return (
        <div className="flex flex-col gap-2 pb-3">
            <Rows
                entries={[
                    [copy.runs.review.publish.result, view.created ? copy.runs.review.publish.created : copy.runs.review.publish.updated],
                    [copy.runs.review.publish.status, view.status],
                    [copy.runs.review.publish.wpId, view.wpId === null ? "" : String(view.wpId)],
                    [copy.runs.review.publish.hash, view.contentHash],
                    [copy.runs.review.publish.seoApplied, view.seoApplied.join(", ")],
                    [copy.runs.review.publish.skipped, view.skipped.join(", ")],
                ]}
            />
            {view.url === "" ? null : (
                <div className="flex min-w-0 flex-col gap-0.5 px-3">
                    <SectionLabel>{copy.runs.review.publish.liveUrl}</SectionLabel>
                    <ExternalUrl url={view.url} />
                </div>
            )}
            <p className="px-3 text-2xs text-ink-faint">{copy.runs.review.noDiff}</p>
            <FindingTotals findings={view.findings} />
            <FindingList findings={view.findings} empty={copy.runs.review.links.clean} />
        </div>
    );
}

interface PayloadPaneProps {
    payload: unknown;
}

function DraftPane({ payload }: PayloadPaneProps): ReactElement {
    const view = draftView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    return (
        <div className="flex flex-col gap-2 pb-3">
            <Rows
                entries={[
                    [copy.pages.detail.title, view.title],
                    [copy.pages.detail.h1, view.h1],
                    [copy.runs.review.draft.summary, view.summary],
                ]}
            />
            <SectionLabel className="px-3">{copy.runs.review.draft.sections}</SectionLabel>
            <ul className="flex flex-col">
                {view.sections.map((section, position) => (
                    <li
                        key={`${section.heading}:${String(position)}`}
                        className="border-b border-inset px-3 py-1.5 text-xs text-ink-soft last:border-b-0"
                    >
                        {section.heading}
                    </li>
                ))}
            </ul>
        </div>
    );
}

function MetaPane({ payload }: PayloadPaneProps): ReactElement {
    const view = metaView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    return (
        <Rows
            entries={[
                [copy.pages.detail.metaTitle, view.title],
                [copy.pages.detail.metaDescription, view.description],
                [copy.pages.detail.canonicalUrl, view.canonical],
                ["og:title", view.ogTitle],
                ["og:description", view.ogDescription],
            ]}
        />
    );
}

function ImagesPane({ payload }: PayloadPaneProps): ReactElement {
    const view = imagesView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    if (view.images.length === 0) {
        return <p className="px-3 py-2 text-xs text-ink-dim">{copy.runs.review.images.none}</p>;
    }
    return (
        <div className="flex flex-col gap-2 pb-3">
            <ul className="flex flex-col">
                {view.images.map((image) => (
                    <li
                        key={image.url}
                        className="flex items-center gap-2 border-b border-inset px-3 py-1.5 text-xs last:border-b-0"
                    >
                        <span className="w-20 shrink-0 text-2xs text-ink-faint">{image.role}</span>
                        <span className="min-w-0 flex-1 truncate font-mono text-ink-soft">{image.url}</span>
                        <span className="w-40 shrink-0 truncate text-2xs text-ink-faint">{image.alt}</span>
                    </li>
                ))}
            </ul>
            <Rows
                entries={[
                    [copy.runs.review.images.featured, view.featuredId === null ? "" : String(view.featuredId)],
                    [copy.runs.review.images.skipped, view.skipped.join(", ")],
                ]}
            />
        </div>
    );
}

function RelinkPane({ payload }: PayloadPaneProps): ReactElement {
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

function SyncPane({ payload }: PayloadPaneProps): ReactElement {
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
            {view.url === "" ? null : (
                <div className="flex min-w-0 flex-col gap-0.5 px-3">
                    <SectionLabel>{copy.runs.review.publish.liveUrl}</SectionLabel>
                    <ExternalUrl url={view.url} />
                </div>
            )}
        </div>
    );
}

function FinalPane({ payload }: PayloadPaneProps): ReactElement {
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

interface BodyPaneProps {
    html: string;
}

function BodyPane({ html }: BodyPaneProps): ReactElement {
    if (html === "") {
        return <p className="px-3 py-2 text-xs text-ink-dim">{copy.runs.review.body.empty}</p>;
    }
    return (
        <div className="flex h-full min-h-0 flex-col">
            <div className="flex shrink-0 items-center justify-between gap-2 px-3 py-1.5 text-2xs text-ink-faint">
                <span>{copy.runs.review.body.preview}</span>
                <span>{copy.runs.review.body.isolated}</span>
            </div>
            <div className="h-[32rem] min-h-0 shrink-0">
                <HtmlPreview bodyHtml={html} title={copy.runs.review.body.preview} />
            </div>
        </div>
    );
}

export interface ArtifactPaneProps {
    itemId: string;
    kind: ArtifactKind;
    pageId: string;
    retentionDays: number | null;
}

export function ArtifactPane({ itemId, kind, pageId, retentionDays }: ArtifactPaneProps): ReactElement {
    const artifact = useArtifact(itemId, kind);

    if (artifact.isPending) {
        return (
            <div className="p-3">
                <SkeletonRows rows={6} label={copy.runs.review.loading} />
            </div>
        );
    }

    if (artifact.data === undefined) {
        const reported = artifact.isError ? failure(artifact.error) : null;
        return (
            <div className="p-3">
                <EmptyState
                    icon={TaskAltIcon}
                    title={copy.runs.review.noArtifacts}
                    body={reported === null ? copy.runs.review.noArtifactsBody : reported.message}
                />
            </div>
        );
    }

    const row = artifact.data.artifact;

    if (row.purged) {
        return (
            <div className="flex flex-col gap-2 p-3">
                <Banner
                    tone="info"
                    icon={retentionIcon}
                    title={copy.runs.review.purged}
                    body={
                        retentionDays === null
                            ? copy.runs.review.purgedBodyUnknown
                            : copy.runs.review.purgedBody(retentionDays)
                    }
                />
                <p className="font-mono text-2xs text-ink-faint" title={absoluteTime(row.createdAt)}>
                    {`${row.step} · ${relativeTime(row.createdAt)}`}
                </p>
            </div>
        );
    }

    if (kind === artifactBodyHtml) {
        return <BodyPane html={row.content} />;
    }

    const payload = decodeArtifact(row.content);

    return (
        <div className="flex min-h-0 flex-col">
            <div className="flex shrink-0 items-center justify-between gap-2 border-b border-inset px-3 py-1 font-mono text-2xs text-ink-faint">
                <span title={absoluteTime(row.createdAt)}>{`${row.step} · ${relativeTime(row.createdAt)}`}</span>
                <span>{bytes(row.size)}</span>
            </div>
            {kind === artifactLinkContext ? (
                <LinkContextPane context={linkContextView(payload)} />
            ) : kind === artifactDraft ? (
                <DraftPane payload={payload} />
            ) : kind === artifactMeta ? (
                <MetaPane payload={payload} />
            ) : kind === artifactImages ? (
                <ImagesPane payload={payload} />
            ) : kind === artifactValidationReport ? (
                <ValidationPane itemId={itemId} payload={payload} />
            ) : kind === artifactJudgeReport ? (
                <JudgePane payload={payload} pageId={pageId} />
            ) : kind === artifactPublishResult ? (
                <PublishPane payload={payload} />
            ) : kind === artifactRelinkResult ? (
                <RelinkPane payload={payload} />
            ) : kind === artifactSyncResult ? (
                <SyncPane payload={payload} />
            ) : kind === artifactFinalReport ? (
                <FinalPane payload={payload} />
            ) : (
                <Unreadable />
            )}
        </div>
    );
}
