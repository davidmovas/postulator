import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { failure } from "../../../data/errors.js";
import { useArtifact } from "../../../data/hooks/runs.js";
import { absoluteTime, bytes, relativeTime } from "../../../domain/format.js";
import type { ArtifactKind } from "../../../generated/vocab.js";
import { Banner, EmptyState, SkeletonRows, TaskAltIcon } from "../../../ui/index.js";
import { decodeArtifact, linkContextView } from "../artifacts.js";
import { retentionIcon } from "../labels.js";
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
} from "../statuses.js";
import { BodyPane, DraftPane, ImagesPane, MetaPane } from "./content.js";
import { JudgePane } from "./judge.js";
import { LinkContextPane, ValidationPane } from "./links.js";
import { FinalPane, PublishPane, RelinkPane, SyncPane } from "./publish.js";
import { Unreadable } from "./shared.js";

interface PayloadProps {
    itemId: string;
    kind: ArtifactKind;
    pageId: string;
    payload: unknown;
}

function Payload({ itemId, kind, pageId, payload }: PayloadProps): ReactElement {
    switch (kind) {
        case artifactLinkContext:
            return <LinkContextPane context={linkContextView(payload)} />;
        case artifactDraft:
            return <DraftPane payload={payload} />;
        case artifactMeta:
            return <MetaPane payload={payload} />;
        case artifactImages:
            return <ImagesPane payload={payload} />;
        case artifactValidationReport:
            return <ValidationPane itemId={itemId} payload={payload} />;
        case artifactJudgeReport:
            return <JudgePane payload={payload} pageId={pageId} />;
        case artifactPublishResult:
            return <PublishPane payload={payload} />;
        case artifactRelinkResult:
            return <RelinkPane payload={payload} />;
        case artifactSyncResult:
            return <SyncPane payload={payload} />;
        case artifactFinalReport:
            return <FinalPane payload={payload} />;
        default:
            return <Unreadable />;
    }
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

    return (
        <div className="flex min-h-0 flex-col">
            <div className="flex shrink-0 items-center justify-between gap-2 border-b border-inset px-3 py-1 font-mono text-2xs text-ink-faint">
                <span title={absoluteTime(row.createdAt)}>{`${row.step} · ${relativeTime(row.createdAt)}`}</span>
                <span>{bytes(row.size)}</span>
            </div>
            <Payload itemId={itemId} kind={kind} pageId={pageId} payload={decodeArtifact(row.content)} />
        </div>
    );
}
