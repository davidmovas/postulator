import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { useApproveEdge, useRejectEdge } from "../../../data/hooks/graph.js";
import { react } from "../../../data/errors.js";
import { pushToast } from "../../../data/toasts.js";
import type { Edge } from "../../../data/types.js";
import { Dialog, ProgressBar } from "../../../ui/index.js";

export type BulkDecision = "approve" | "reject";

export interface BulkRequest {
    decision: BulkDecision;
    threshold: number;
    edges: readonly Edge[];
}

export interface BulkDialogProps {
    request: BulkRequest | null;
    onClose: () => void;
}

export function BulkDialog({ request, onClose }: BulkDialogProps): ReactElement {
    const approve = useApproveEdge();
    const reject = useRejectEdge();
    const [done, setDone] = useState(0);
    const [running, setRunning] = useState(false);

    const run = async (): Promise<void> => {
        if (request === null) {
            return;
        }
        setRunning(true);
        setDone(0);
        let settled = 0;
        try {
            for (const edge of request.edges) {
                if (request.decision === "approve") {
                    await approve.mutateAsync({ id: edge.id });
                } else {
                    await reject.mutateAsync({ id: edge.id });
                }
                settled += 1;
                setDone(settled);
            }
            pushToast("info", copy.graph.queue.bulkDone(settled, request.decision));
        } catch (thrown) {
            const reaction = react(thrown);
            if (reaction.kind !== "silent" && reaction.kind !== "unlock") {
                pushToast("danger", copy.graph.queue.bulkStopped(settled, request.edges.length, reaction.message));
            }
        } finally {
            setRunning(false);
            onClose();
        }
    };

    const title =
        request === null
            ? ""
            : request.decision === "approve"
              ? copy.graph.queue.bulkApproveTitle(request.edges.length, request.threshold)
              : copy.graph.queue.bulkRejectTitle(request.edges.length, request.threshold);

    return (
        <Dialog
            open={request !== null}
            onOpenChange={(open) => {
                if (!open && !running) {
                    onClose();
                }
            }}
            title={title}
            description={copy.graph.queue.bulkBody}
            confirmLabel={copy.graph.queue.confirm}
            cancelLabel={copy.graph.queue.cancel}
            destructive={request?.decision === "reject"}
            busy={running}
            onConfirm={() => {
                void run();
            }}
        >
            {running && request !== null ? (
                <ProgressBar
                    value={done}
                    max={request.edges.length}
                    label={copy.graph.queue.progress(done, request.edges.length)}
                    trailing={copy.graph.queue.progress(done, request.edges.length)}
                />
            ) : null}
        </Dialog>
    );
}
