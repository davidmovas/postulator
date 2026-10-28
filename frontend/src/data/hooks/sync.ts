import { useMutation, useQueryClient } from "@tanstack/react-query";

import { activeRunStatuses } from "../../generated/vocab.js";
import { flatten } from "../call.js";
import { quietMeta } from "../client.js";
import { checkPlugin, savePluginPackage, syncProgress, syncSite } from "../endpoints/sync.js";
import { keys } from "../keys.js";
import { useUnlockedQuery } from "../query.js";
import { catchUpNow, ensureLog } from "../runs/log.js";
import type { RunSort } from "../sorts.js";
import type { Run, SyncProgress } from "../types.js";
import { useRuns } from "./runs.js";

export const syncPollMs = 4000;

const syncKind = "sync";

const newestFirst: RunSort = { field: "createdAt", desc: true };

export function syncUnderWay(progress: SyncProgress | undefined): progress is SyncProgress {
    return (
        progress !== undefined && progress.found && (activeRunStatuses as readonly string[]).includes(progress.status)
    );
}

export function syncPoll(progress: SyncProgress | undefined): number | false {
    return syncUnderWay(progress) ? syncPollMs : false;
}

function useProgress(siteId: string, runId: string, enabled: boolean) {
    return useUnlockedQuery({
        queryKey: keys.sync.progress(siteId, runId),
        queryFn: ({ signal }) => syncProgress(runId === "" ? { siteId } : { siteId, runId }, signal),
        enabled: enabled && siteId !== "",
        refetchInterval: (query) => syncPoll(query.state.data),
        quiet: ["NOT_FOUND", "INVALID"],
    });
}

export function useSyncProgress(siteId: string | null, runId?: string) {
    return useProgress(siteId ?? "", runId ?? "", true);
}

export function useActiveSync(siteId: string | null): SyncProgress | null {
    const progress = useSyncProgress(siteId);
    return syncUnderWay(progress.data) ? progress.data : null;
}

export interface LatestSync {
    run: Run | undefined;
    progress: SyncProgress | undefined;
}

export function useLatestSync(siteId: string): LatestSync {
    const listed = useRuns({ siteId, kind: syncKind }, newestFirst, 1);
    const run = flatten(listed.data?.pages)[0];
    const progress = useProgress(siteId, run?.id ?? "", run !== undefined);
    return { run, progress: run === undefined ? undefined : progress.data };
}

export function usePluginState(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.sync.plugin(siteId ?? ""),
        queryFn: ({ signal }) => checkPlugin({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

export function useSyncSite() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof syncSite>[0]) => syncSite(request),
        meta: quietMeta(["CONFLICT"]),
        onSuccess: (answered) => {
            ensureLog(answered.runId);
            void catchUpNow(answered.runId);
            void client.invalidateQueries({ queryKey: keys.runs.lists() });
        },
        onSettled: () => {
            void client.invalidateQueries({ queryKey: keys.sync.progresses() });
        },
    });
}

export function useSavePluginPackage() {
    return useMutation({
        mutationFn: (request: Parameters<typeof savePluginPackage>[0]) => savePluginPackage(request),
    });
}
