import { useNavigate } from "react-router";

import { useActiveSync, useSyncSite } from "../../data/hooks/sync.js";
import { pushToast } from "../../data/toasts.js";
import type { SyncProgress, SyncSiteRequest } from "../../data/types.js";
import { syncConflictOf, syncRunPath } from "./model.js";

export type SyncOptions = Omit<SyncSiteRequest, "siteId">;

export interface StartSync {
    active: SyncProgress | null;
    pending: boolean;
    start: (options: SyncOptions, onStarted?: (runId: string) => void) => void;
    open: (runId: string) => void;
}

export function useStartSync(siteId: string): StartSync {
    const navigate = useNavigate();
    const sync = useSyncSite();
    const active = useActiveSync(siteId === "" ? null : siteId);

    const open = (runId: string): void => {
        void navigate(syncRunPath(siteId, runId));
    };

    const start = (options: SyncOptions, onStarted?: (runId: string) => void): void => {
        sync.mutate(
            { siteId, ...options },
            {
                onSuccess: (answered) => {
                    onStarted?.(answered.runId);
                },
                onError: (thrown) => {
                    const conflict = syncConflictOf(thrown);
                    if (conflict === null) {
                        return;
                    }
                    pushToast("info", conflict.message);
                    if (conflict.runId !== null) {
                        open(conflict.runId);
                    }
                },
            },
        );
    };

    return { active, pending: sync.isPending, start, open };
}
