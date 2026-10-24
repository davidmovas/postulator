import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { useSyncSite } from "../../data/hooks/sync.js";
import type { SyncSiteRequest } from "../../data/types.js";
import { Checkbox, Dialog, SyncIcon } from "../../ui/index.js";

const said = copy.sites.sync;

export interface SyncDialogProps {
    request: SyncSiteRequest | null;
    onOpenChange: (open: boolean) => void;
    onStarted: (runId: string) => void;
}

export function SyncDialog({ request, onOpenChange, onStarted }: SyncDialogProps): ReactElement {
    const sync = useSyncSite();
    const [full, setFull] = useState(false);
    const [renderRescan, setRenderRescan] = useState(false);

    useEffect(() => {
        setFull(request?.full === true);
        setRenderRescan(request?.renderRescan === true);
    }, [request]);

    return (
        <Dialog
            open={request !== null}
            onOpenChange={onOpenChange}
            title={said.againTitle}
            description={said.againBody}
            confirmLabel={said.start}
            cancelLabel={said.cancel}
            icon={SyncIcon}
            busy={sync.isPending}
            onConfirm={() => {
                if (request === null) {
                    return;
                }
                sync.mutate(
                    { siteId: request.siteId, full, renderRescan },
                    {
                        onSuccess: (answered) => {
                            onStarted(answered.runId);
                        },
                    },
                );
            }}
        >
            <span title={said.fullTooltip} className="w-fit">
                <Checkbox
                    label={said.full}
                    checked={full}
                    onChange={(event) => {
                        setFull(event.currentTarget.checked);
                    }}
                />
            </span>
            <span title={said.rescanTooltip} className="w-fit">
                <Checkbox
                    label={said.rescan}
                    checked={renderRescan}
                    onChange={(event) => {
                        setRenderRescan(event.currentTarget.checked);
                    }}
                />
            </span>
        </Dialog>
    );
}
