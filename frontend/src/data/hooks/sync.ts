import { useMutation, useQueryClient } from "@tanstack/react-query";

import { checkPlugin, savePluginPackage, syncSite } from "../endpoints/sync.js";
import { keys } from "../keys.js";
import { useUnlockedQuery } from "../query.js";
import { catchUpNow, ensureLog } from "../runs/log.js";

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
        onSuccess: (answered) => {
            ensureLog(answered.runId);
            void catchUpNow(answered.runId);
            void client.invalidateQueries({ queryKey: keys.runs.lists() });
        },
    });
}

export function useSavePluginPackage() {
    return useMutation({
        mutationFn: (request: Parameters<typeof savePluginPackage>[0]) => savePluginPackage(request),
    });
}
