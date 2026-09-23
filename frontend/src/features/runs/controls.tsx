import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { useCancelRun, usePauseRun, useResumeRun, useRevertRun } from "../../data/hooks/runs.js";
import type { Run } from "../../data/types.js";
import { Button, CancelIcon, Dialog, HistoryIcon, PauseIcon, PlayArrowIcon } from "../../ui/index.js";
import type { RevertBlock } from "./authority.js";
import { revertState, runView } from "./authority.js";
import { statusRunning } from "./statuses.js";

export interface RunControlsProps {
    run: Run;
    onReverted?: (runId: string) => void;
}

const revertBlocks: Readonly<Record<RevertBlock, string>> = {
    in_flight: copy.runs.revertRunning,
    already_a_revert: copy.runs.revertOfARevert,
};

export function RunControls({ run, onReverted }: RunControlsProps): ReactElement {
    const view = runView(run);
    const revertable = revertState(run);
    const pause = usePauseRun();
    const resume = useResumeRun();
    const cancel = useCancelRun();
    const revert = useRevertRun();
    const [confirming, setConfirming] = useState(false);
    const [reverting, setReverting] = useState(false);

    const pausable = run.status === statusRunning;
    const blocked = (why: string | undefined): string | undefined =>
        view.terminal ? copy.runs.settled : why;

    return (
        <>
            <Button
                data-run-pause={true}
                icon={PauseIcon}
                disabled={!pausable}
                busy={pause.isPending}
                title={blocked(view.paused ? copy.runs.alreadyPaused : copy.runs.notRunning)}
                onClick={() => {
                    pause.mutate({ runId: run.id });
                }}
            >
                {copy.runs.pause}
            </Button>
            <Button
                icon={PlayArrowIcon}
                disabled={!view.paused}
                busy={resume.isPending}
                title={view.paused ? undefined : blocked(copy.runs.notPaused)}
                onClick={() => {
                    resume.mutate({ runId: run.id });
                }}
            >
                {copy.runs.resume}
            </Button>
            <Button
                data-run-revert={true}
                icon={HistoryIcon}
                disabled={revertable.kind !== "ready"}
                busy={revert.isPending}
                title={revertable.kind === "ready" ? undefined : revertBlocks[revertable.reason]}
                onClick={() => {
                    setReverting(true);
                }}
            >
                {copy.runs.revert}
            </Button>
            <Button
                variant="danger"
                icon={CancelIcon}
                disabled={view.terminal}
                title={view.terminal ? copy.runs.settled : undefined}
                onClick={() => {
                    setConfirming(true);
                }}
            >
                {copy.runs.cancel}
            </Button>
            <Dialog
                open={confirming}
                onOpenChange={setConfirming}
                title={copy.runs.cancelTitle}
                description={copy.runs.cancelBody}
                icon={CancelIcon}
                destructive={true}
                busy={cancel.isPending}
                confirmLabel={copy.runs.cancelConfirm}
                cancelLabel={copy.runs.keepRunning}
                onConfirm={() => {
                    cancel.mutate(
                        { runId: run.id },
                        {
                            onSuccess: () => {
                                setConfirming(false);
                            },
                        },
                    );
                }}
            />
            <Dialog
                open={reverting}
                onOpenChange={setReverting}
                title={copy.runs.revertTitle}
                description={copy.runs.revertBody}
                icon={HistoryIcon}
                destructive={true}
                busy={revert.isPending}
                confirmLabel={copy.runs.revertConfirm}
                cancelLabel={copy.runs.revertKeep}
                onConfirm={() => {
                    revert.mutate(
                        { runId: run.id },
                        {
                            onSuccess: (answered) => {
                                setReverting(false);
                                onReverted?.(answered.runId);
                            },
                        },
                    );
                }}
            />
        </>
    );
}
