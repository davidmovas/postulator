import { useMutation, useQueryClient } from "@tanstack/react-query";

import { judgePage, linkAudit, linkAuditPage, pageReport, runReport, siteOverview } from "../endpoints/reports.js";
import { keys } from "../keys.js";
import { useUnlockedQuery } from "../query.js";

export const linkAuditStaleMs = 30_000;

export function useLinkAudit(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.reports.siteLinks(siteId ?? ""),
        queryFn: ({ signal }) => linkAudit({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
        staleTime: linkAuditStaleMs,
        refetchOnWindowFocus: false,
    });
}

export function useLinkAuditPage(pageId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.reports.pageLinks(pageId ?? ""),
        queryFn: ({ signal }) => linkAuditPage({ pageId: pageId ?? "" }, signal),
        enabled: pageId !== null && pageId !== "",
        quiet: ["NOT_FOUND"],
    });
}

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
