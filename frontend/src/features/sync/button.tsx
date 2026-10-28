import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { SyncProgress } from "../../data/types.js";
import type { ButtonSize, ButtonVariant, IconComponent } from "../../ui/index.js";
import { Button, SyncIcon } from "../../ui/index.js";
import { stagePhrase, syncLine } from "./model.js";
import type { SyncOptions } from "./start.js";
import { useStartSync } from "./start.js";

const said = copy.sync;

const noOptions: SyncOptions = {};

export interface SyncUnderWayProps {
    progress: SyncProgress;
    size?: ButtonSize;
    onOpen: (runId: string) => void;
}

export function SyncUnderWay({ progress, size = "md", onOpen }: SyncUnderWayProps): ReactElement {
    return (
        <span className="inline-flex min-w-0 items-center gap-1" data-sync-under-way={progress.runId}>
            <Button size={size} busy={true} title={syncLine(progress)} className="min-w-0">
                <span className="truncate">{said.syncing(stagePhrase(progress))}</span>
            </Button>
            <Button
                size={size}
                variant="ghost"
                data-sync-open={progress.runId}
                onClick={() => {
                    onOpen(progress.runId);
                }}
            >
                {said.open}
            </Button>
        </span>
    );
}

export interface SyncButtonProps {
    siteId: string;
    label: string;
    options?: SyncOptions;
    onStarted?: (runId: string) => void;
    variant?: ButtonVariant;
    size?: ButtonSize;
    icon?: IconComponent | null;
    title?: string;
}

export function SyncButton({
    siteId,
    label,
    options = noOptions,
    onStarted,
    variant = "secondary",
    size = "md",
    icon = SyncIcon,
    title,
}: SyncButtonProps): ReactElement {
    const sync = useStartSync(siteId);

    if (sync.active !== null) {
        return <SyncUnderWay progress={sync.active} size={size} onOpen={sync.open} />;
    }

    return (
        <Button
            variant={variant}
            size={size}
            icon={icon ?? undefined}
            busy={sync.pending}
            title={title}
            data-sync-start={siteId}
            onClick={() => {
                sync.start(options, onStarted);
            }}
        >
            {label}
        </Button>
    );
}
