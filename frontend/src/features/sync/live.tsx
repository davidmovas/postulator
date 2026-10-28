import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { react } from "../../data/errors.js";
import { useRegenerate, useRetryStep, useRunItems } from "../../data/hooks/runs.js";
import { pushToast } from "../../data/toasts.js";
import type { SyncProgress } from "../../data/types.js";
import { Button, cx, ErrorIcon, PlayArrowIcon, RestartAltIcon } from "../../ui/index.js";
import type { StageState, StageView } from "./model.js";
import { stageSteps, stoppedTitle, syncLine } from "./model.js";

const said = copy.sync;

const failedStatus = "failed";
const failedPageSize = 5;

const segmentTone: Readonly<Record<StageState, string>> = {
    done: "bg-ink-dim",
    current: "bg-accent",
    failed: "bg-danger",
    stopped: "bg-muted",
    pending: "bg-raised",
};

export interface SyncMeterProps {
    progress: StageView;
    className?: string;
}

export function SyncMeter({ progress, className }: SyncMeterProps): ReactElement {
    const steps = stageSteps(progress);
    return (
        <span role="img" aria-label={said.stagesLabel} className={cx("flex h-1 w-full gap-0.5", className)}>
            {steps.map((step) => (
                <span
                    key={step.stage}
                    data-stage-state={step.state}
                    title={step.words}
                    className={cx("h-full min-w-0 flex-1 rounded-sm", segmentTone[step.state])}
                />
            ))}
        </span>
    );
}

export interface SyncLiveProps {
    progress: SyncProgress;
}

export function SyncLive({ progress }: SyncLiveProps): ReactElement {
    const line = syncLine(progress);
    return (
        <div className="flex min-w-0 flex-col gap-1.5" data-sync-live={progress.runId}>
            <SyncMeter progress={progress} />
            <p className="truncate text-xs text-ink-soft" title={line}>
                {line}
            </p>
        </div>
    );
}

export interface SyncErrorProps {
    error: string;
    compact?: boolean;
    className?: string;
}

export function SyncError({ error, compact = false, className }: SyncErrorProps): ReactElement {
    return (
        <pre
            data-sync-error={true}
            className={cx(
                "min-w-0 rounded-sm border border-hairline bg-inset px-2 py-1.5 font-mono text-2xs leading-relaxed whitespace-pre-wrap text-ink select-text wrap-anywhere",
                compact ? "overflow-hidden" : "max-h-48 overflow-y-auto",
                className,
            )}
        >
            {compact ? (
                <span title={error} className="line-clamp-4">
                    {error}
                </span>
            ) : (
                error
            )}
        </pre>
    );
}

function refusalOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "silent" || reaction.kind === "unlock" ? null : reaction.message;
}

export interface SyncStoppedProps {
    runId: string;
    stage: string;
    error: string;
    title?: string;
    startOver?: boolean;
    compact?: boolean;
    onOpen?: (runId: string) => void;
}

const fitted = "min-w-0 max-w-full";

export function SyncStopped({
    runId,
    stage,
    error,
    title,
    startOver = false,
    compact = false,
    onOpen,
}: SyncStoppedProps): ReactElement {
    const listed = useRunItems(runId, failedStatus, failedPageSize);
    const retry = useRetryStep();
    const regenerate = useRegenerate();
    const itemIds = useMemo(() => flatten(listed.data?.pages).map((item) => item.id), [listed.data]);
    const first = itemIds[0];
    const refusal = refusalOf(retry.error) ?? refusalOf(regenerate.error);
    const settling = listed.isPending || retry.isPending || regenerate.isPending;

    return (
        <div
            role="note"
            data-sync-stopped={runId}
            className="flex min-w-0 flex-col gap-2 rounded-lg border border-danger-border bg-danger-soft p-3"
        >
            <p className="flex min-w-0 items-start gap-2 text-sm font-semibold text-danger">
                <ErrorIcon size={17} className="mt-px shrink-0" />
                <span className="min-w-0">{title ?? stoppedTitle(stage)}</span>
            </p>
            {error === "" ? null : <SyncError error={error} compact={compact} />}
            <p className="text-xs text-ink-soft">
                {refusal ?? (startOver ? said.stopped.body : said.stopped.resumeBody)}
            </p>
            <div className="flex min-w-0 flex-wrap gap-2">
                <Button
                    size="sm"
                    variant="primary"
                    icon={PlayArrowIcon}
                    className={fitted}
                    data-sync-resume={runId}
                    disabled={first === undefined}
                    busy={settling}
                    onClick={() => {
                        if (first === undefined) {
                            return;
                        }
                        retry.mutate(
                            { itemId: first },
                            {
                                onSuccess: () => {
                                    pushToast("info", said.resumed);
                                },
                            },
                        );
                    }}
                >
                    <span className="min-w-0 truncate">{said.resume}</span>
                </Button>
                {startOver ? (
                    <Button
                        size="sm"
                        icon={RestartAltIcon}
                        className={fitted}
                        data-sync-start-over={runId}
                        disabled={itemIds.length === 0 || settling}
                        onClick={() => {
                            regenerate.mutate({ runId, itemIds });
                        }}
                    >
                        <span className="min-w-0 truncate">{said.startOver}</span>
                    </Button>
                ) : null}
                {onOpen === undefined ? null : (
                    <Button
                        size="sm"
                        variant="ghost"
                        className={fitted}
                        onClick={() => {
                            onOpen(runId);
                        }}
                    >
                        <span className="min-w-0 truncate">{said.open}</span>
                    </Button>
                )}
            </div>
        </div>
    );
}
