import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { absoluteTime, duration, relativeTime } from "../../domain/format.js";
import {
    Banner,
    Button,
    cx,
    EmptyState,
    HourglassEmptyIcon,
    Kbd,
    OpenInNewIcon,
    RefreshIcon,
    RestartAltIcon,
    SmartToyIcon,
    SyncProblemIcon,
} from "../../ui/index.js";
import { askAgent } from "../agent/index.js";
import type { RetryState } from "./authority.js";
import { countdown } from "./authority.js";
import type { RegenerateState } from "./hold.js";
import { retryBlockedText, stepLabel } from "./labels.js";
import type { RetryNotice, StepEntry } from "./log-view.js";
import type { DriftRefusal } from "./refusal.js";

export function Timeline({ entries }: { entries: readonly StepEntry[] }): ReactElement | null {
    if (entries.length === 0) {
        return null;
    }
    return (
        <div className="flex flex-wrap items-center gap-1 border-b border-hairline px-3 py-1.5">
            {entries.map((entry, position) => (
                <span
                    key={`${entry.step}:${String(position)}`}
                    title={entry.message ?? undefined}
                    className={cx(
                        "inline-flex items-center gap-1 rounded-sm px-1.5 py-0.5 font-mono text-2xs",
                        entry.code !== null
                            ? "bg-danger-soft text-danger"
                            : entry.finishedAt === null
                              ? "bg-info-soft text-info"
                              : "bg-inset text-ink-dim",
                    )}
                >
                    {stepLabel(entry.step)}
                    {entry.durationMs === null ? null : (
                        <span className="text-ink-faint">{duration(entry.durationMs)}</span>
                    )}
                    {entry.attempts > 1 ? (
                        <span className="text-warn">{copy.runs.events.attempt(entry.attempts)}</span>
                    ) : null}
                </span>
            ))}
        </div>
    );
}

export interface ReviewMetaProps {
    step: string;
    attempts: number;
    wake: number | null;
    retry: RetryNotice | undefined;
    retryLeft: number | null;
}

export function ReviewMeta({ step, attempts, wake, retry, retryLeft }: ReviewMetaProps): ReactElement {
    return (
        <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-3 py-2">
            <span className="font-mono text-2xs text-ink-faint">{step}</span>
            <span className="font-mono text-2xs text-ink-faint">{copy.runs.review.attempt(attempts)}</span>
            {wake === null ? null : (
                <span className="font-mono text-2xs text-info">
                    {copy.runs.step.waitingUntil(countdown(wake))}
                </span>
            )}
            {retry === undefined || retryLeft === null ? null : (
                <span className="font-mono text-2xs text-warn">
                    {copy.runs.step.retrying(retry.attempt, countdown(retryLeft))}
                </span>
            )}
        </div>
    );
}

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

export interface ReviewActionsProps {
    path: string;
    runId: string;
    itemId: string;
    pageId: string;
    blocked: string | null;
    state: RetryState | null;
    busy: boolean;
    regeneration: RegenerateState | null;
    regenerating: boolean;
    onOpenPage: (pageId: string) => void;
    onRerun: (pageId: string) => void;
    onRetry: () => void;
    onRegenerate: () => void;
}

const regenerateRefusals: Readonly<Record<Exclude<RegenerateState["kind"], "ready">, string>> = {
    busy: copy.runs.regenerateBusy,
    published: copy.runs.regeneratePublished,
};

export function ReviewActions({
    path,
    runId,
    itemId,
    pageId,
    blocked,
    state,
    busy,
    regeneration,
    regenerating,
    onOpenPage,
    onRerun,
    onRetry,
    onRegenerate,
}: ReviewActionsProps): ReactElement {
    return (
        <div className="flex w-full items-center justify-between gap-2">
            <span className="flex shrink-0 items-center gap-1.5 text-2xs text-ink-faint">
                <Kbd keys={["J", "K"]} />
                {copy.runs.review.moveItems}
            </span>
            <div className="flex shrink-0 gap-2">
                <Button
                    size="sm"
                    icon={OpenInNewIcon}
                    onClick={() => {
                        onOpenPage(pageId);
                    }}
                >
                    {copy.runs.review.openPage}
                </Button>
                <Button
                    size="sm"
                    variant="ghost"
                    icon={SmartToyIcon}
                    onClick={() => {
                        askAgent(copy.agent.ask.runItem(path, runId, itemId));
                    }}
                >
                    {copy.agent.askAbout}
                </Button>
                {blocked === null && regeneration?.kind !== "published" ? null : (
                    <Button
                        size="sm"
                        onClick={() => {
                            onRerun(pageId);
                        }}
                    >
                        {copy.runs.rerunPage}
                    </Button>
                )}
                <Button
                    size="sm"
                    data-item-regenerate={true}
                    icon={RefreshIcon}
                    disabled={regeneration === null || regeneration.kind !== "ready"}
                    busy={regenerating}
                    title={
                        regeneration === null || regeneration.kind === "ready"
                            ? copy.runs.regenerateHint
                            : regenerateRefusals[regeneration.kind]
                    }
                    onClick={onRegenerate}
                >
                    {copy.runs.regenerate}
                </Button>
                <Button
                    size="sm"
                    variant="primary"
                    icon={RestartAltIcon}
                    disabled={state === null || state.kind !== "ready"}
                    busy={busy}
                    title={
                        state !== null && state.kind === "busy"
                            ? copy.runs.retryBusy
                            : blocked === null
                              ? undefined
                              : retryBlockedText(blocked)
                    }
                    onClick={onRetry}
                >
                    {copy.runs.retryStep}
                </Button>
            </div>
        </div>
    );
}
