import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import type { SyncSiteRequest } from "../../data/types.js";
import { Checkbox, Dialog, SyncIcon } from "../../ui/index.js";
import { SyncLive, useStartSync } from "../sync/index.js";

const said = copy.sites.sync;

export interface SyncDialogProps {
    request: SyncSiteRequest | null;
    onOpenChange: (open: boolean) => void;
    onStarted: (runId: string) => void;
}

export function SyncDialog({ request, onOpenChange, onStarted }: SyncDialogProps): ReactElement {
    const sync = useStartSync(request?.siteId ?? "");
    const [full, setFull] = useState(false);
    const [renderRescan, setRenderRescan] = useState(false);

    useEffect(() => {
        setFull(request?.full === true);
        setRenderRescan(request?.renderRescan === true);
    }, [request]);

    if (sync.active !== null) {
        const active = sync.active;
        return (
            <Dialog
                open={request !== null}
                onOpenChange={onOpenChange}
                title={copy.sync.underWay.title}
                description={copy.sync.underWay.body}
                confirmLabel={copy.sync.open}
                cancelLabel={said.cancel}
                icon={SyncIcon}
                onConfirm={() => {
                    onStarted(active.runId);
                }}
            >
                <SyncLive progress={active} />
            </Dialog>
        );
    }

    return (
        <Dialog
            open={request !== null}
            onOpenChange={onOpenChange}
            title={said.againTitle}
            description={said.againBody}
            confirmLabel={said.start}
            cancelLabel={said.cancel}
            icon={SyncIcon}
            busy={sync.pending}
            onConfirm={() => {
                if (request === null) {
                    return;
                }
                sync.start({ full, renderRescan }, onStarted);
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
