import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { useCancelRun, usePauseRun, useResumeRun } from "../../data/hooks/runs.js";
import type { Run } from "../../data/types.js";
import { Button, CancelIcon, Dialog, PauseIcon, PlayArrowIcon } from "../../ui/index.js";
import { runView } from "./authority.js";

export interface RunControlsProps {
    run: Run;
}

export function RunControls({ run }: RunControlsProps): ReactElement {
    const view = runView(run);
    const pause = usePauseRun();
    const resume = useResumeRun();
    const cancel = useCancelRun();
    const [confirming, setConfirming] = useState(false);

    return (
        <>
            <Button
                data-run-pause={true}
                icon={PauseIcon}
                disabled={!view.active || view.paused}
                busy={pause.isPending}
                title={view.terminal ? copy.runs.settled : view.paused ? copy.runs.alreadyPaused : undefined}
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
                title={view.paused ? undefined : copy.runs.notPaused}
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
