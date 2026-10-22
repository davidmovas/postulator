import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { failure } from "../../data/errors.js";
import { useRevertRun } from "../../data/hooks/runs.js";
import { pushToast } from "../../data/toasts.js";
import { Dialog, HistoryIcon } from "../../ui/index.js";

const said = copy.runs.revertFields;

export interface RevertFieldsDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    runId: string;
    pageId: string;
    siteId: string;
    fields: readonly string[];
    description: string;
}

export function RevertFieldsDialog({
    open,
    onOpenChange,
    runId,
    pageId,
    siteId,
    fields,
    description,
}: RevertFieldsDialogProps): ReactElement {
    const revert = useRevertRun();
    const refused = revert.error === null ? null : failure(revert.error).message;

    return (
        <Dialog
            open={open}
            onOpenChange={onOpenChange}
            title={said.title}
            description={description}
            icon={HistoryIcon}
            destructive={true}
            busy={revert.isPending}
            status={refused === null ? undefined : <span className="text-xs text-danger">{refused}</span>}
            confirmLabel={said.confirm}
            cancelLabel={copy.runs.revertKeep}
            onConfirm={() => {
                revert.mutate(
                    { runId, pageId, fields: [...fields] },
                    {
                        onSuccess: (answered) => {
                            onOpenChange(false);
                            pushToast(
                                "info",
                                said.started,
                                null,
                                siteId === "" ? null : { label: said.open, to: `/s/${siteId}/runs/${answered.runId}` },
                            );
                        },
                    },
                );
            }}
        />
    );
}
