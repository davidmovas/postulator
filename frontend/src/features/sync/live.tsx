import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { react } from "../../data/errors.js";
import { useRegenerate, useRetryStep, useRunItems } from "../../data/hooks/runs.js";
import { pushToast } from "../../data/toasts.js";
import type { SyncProgress } from "../../data/types.js";
import { Banner, Button, cx, PlayArrowIcon, RestartAltIcon } from "../../ui/index.js";
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
    className?: string;
}

export function SyncError({ error, className }: SyncErrorProps): ReactElement {
    return (
        <pre
            data-sync-error={true}
            className={cx(
                "max-h-48 overflow-y-auto rounded-sm border border-hairline bg-inset px-2 py-1.5 font-mono text-2xs leading-relaxed whitespace-pre-wrap text-ink select-text wrap-anywhere",
                className,
            )}
        >
            {error}
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
    onOpen?: (runId: string) => void;
}

export function SyncStopped({ runId, stage, error, title, startOver = false, onOpen }: SyncStoppedProps): ReactElement {
    const listed = useRunItems(runId, failedStatus, failedPageSize);
    const retry = useRetryStep();
    const regenerate = useRegenerate();
    const itemIds = useMemo(() => flatten(listed.data?.pages).map((item) => item.id), [listed.data]);
    const first = itemIds[0];
    const refusal = refusalOf(retry.error) ?? refusalOf(regenerate.error);
    const settling = listed.isPending || retry.isPending || regenerate.isPending;

    return (
        <Banner
            tone="danger"
            title={title ?? stoppedTitle(stage)}
            body={
                <div className="flex min-w-0 flex-col gap-1.5">
                    {error === "" ? null : <SyncError error={error} />}
                    <p>{refusal ?? (startOver ? said.stopped.body : said.stopped.resumeBody)}</p>
                </div>
            }
            actions={
                <>
                    <Button
                        size="sm"
                        variant="primary"
                        icon={PlayArrowIcon}
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
                        {said.resume}
                    </Button>
                    {startOver ? (
                        <Button
                            size="sm"
                            icon={RestartAltIcon}
                            data-sync-start-over={runId}
                            disabled={itemIds.length === 0 || settling}
                            onClick={() => {
                                regenerate.mutate({ runId, itemIds });
                            }}
                        >
                            {said.startOver}
                        </Button>
                    ) : null}
                    {onOpen === undefined ? null : (
                        <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => {
                                onOpen(runId);
                            }}
                        >
                            {said.open}
                        </Button>
                    )}
                </>
            }
        />
    );
}
