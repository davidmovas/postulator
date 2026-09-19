import { useMutation, useQueryClient } from "@tanstack/react-query";

import { judgePage, pageReport, runReport, siteOverview } from "../endpoints/reports.js";
import { keys } from "../keys.js";
import { useUnlockedQuery } from "../query.js";

export function useSiteOverview(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.reports.site(siteId ?? ""),
        queryFn: ({ signal }) => siteOverview({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

export function usePageReport(pageId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.reports.page(pageId ?? ""),
        queryFn: ({ signal }) => pageReport({ pageId: pageId ?? "" }, signal),
        enabled: pageId !== null && pageId !== "",
        quiet: ["NOT_FOUND"],
    });
}

export function useRunReport(runId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.reports.run(runId ?? ""),
        queryFn: ({ signal }) => runReport({ runId: runId ?? "" }, signal),
        enabled: runId !== null && runId !== "",
    });
}

export function useJudgePage() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof judgePage>[0]) => judgePage(request),
        retry: false,
        onSuccess: (answered, request) => {
            client.setQueryData(keys.reports.judge(request.pageId), answered);
            void client.invalidateQueries({ queryKey: keys.models.usageAll() });
        },
    });
}
