import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { Banner, Button, EmptyState, HourglassEmptyIcon, SyncProblemIcon } from "../../ui/index.js";
import type { DriftRefusal } from "./refusal.js";

export interface ReviewMissingProps {
    narrowed: boolean;
    onClearFilter: () => void;
}

export function ReviewMissing({ narrowed, onClearFilter }: ReviewMissingProps): ReactElement {
    return (
        <div className="p-4">
            <EmptyState
                icon={HourglassEmptyIcon}
                title={copy.runs.notFound}
                body={narrowed ? copy.runs.noItemMatch : copy.empty.runItems}
                actions={narrowed ? <Button onClick={onClearFilter}>{copy.runs.filters.reset}</Button> : undefined}
            />
        </div>
    );
}

export interface ReviewEmptyProps {
    step: string;
}

export function ReviewEmpty({ step }: ReviewEmptyProps): ReactElement {
    return (
        <div className="p-4">
            <EmptyState
                icon={HourglassEmptyIcon}
                title={copy.runs.review.noArtifacts}
                body={copy.runs.review.waitingOn(step)}
            />
        </div>
    );
}

export interface DriftRefusedProps {
    refusal: DriftRefusal;
    onOpenPage: () => void;
    onRerun: () => void;
    onResume: () => void;
    resuming: boolean;
}

export function DriftRefused({
    refusal,
    onOpenPage,
    onRerun,
    onResume,
    resuming,
}: DriftRefusedProps): ReactElement {
    return (
        <div className="shrink-0 px-3 pt-2">
            <Banner
                tone="warn"
                icon={SyncProblemIcon}
                title={copy.runs.review.driftRefused}
                body={
                    <div className="flex flex-col gap-1">
                        <p>{copy.runs.review.driftRefusedBody}</p>
                        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 font-mono text-2xs text-ink-faint">
                            <dt>{copy.pages.drift.lastWritten}</dt>
                            <dd title={absoluteTime(refusal.lastSyncedAt)}>
                                {relativeTime(refusal.lastSyncedAt)}
                            </dd>
                            <dt>{copy.pages.drift.wpModified}</dt>
                            <dd title={absoluteTime(refusal.wpModifiedAt)}>
                                {relativeTime(refusal.wpModifiedAt)}
                            </dd>
                        </dl>
                    </div>
                }
                actions={
                    <>
                        <Button size="sm" onClick={onOpenPage}>
                            {copy.runs.review.openPage}
                        </Button>
                        <Button size="sm" onClick={onRerun}>
                            {copy.runs.rerunPage}
                        </Button>
                        <Button size="sm" variant="ghost" busy={resuming} onClick={onResume}>
                            {copy.runs.resume}
                        </Button>
                    </>
                }
            />
        </div>
    );
}
