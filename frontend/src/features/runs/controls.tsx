import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { useCancelRun, usePauseRun, useResumeRun } from "../../data/hooks/runs.js";
import type { Run } from "../../data/types.js";
import { Button, CancelIcon, Dialog, PauseIcon, PlayArrowIcon } from "../../ui/index.js";
import { runView } from "./authority.js";
import { statusRunning } from "./statuses.js";

export interface RunControlsProps {
    run: Run;
}

export function RunControls({ run }: RunControlsProps): ReactElement {
    const view = runView(run);
    const pause = usePauseRun();
    const resume = useResumeRun();
    const cancel = useCancelRun();
    const [confirming, setConfirming] = useState(false);

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
        </>
    );
}
