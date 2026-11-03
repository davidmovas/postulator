import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
    clearSiteData,
    createSite,
    deleteSite,
    getSite,
    listSites,
    siteDataCounts,
    testConnection,
    updateSite,
} from "../endpoints/sites.js";
import { quietMeta } from "../client.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { SiteSort } from "../sorts.js";
import type { Site, SiteFilter } from "../types.js";

export function useSites(filter: SiteFilter = {}, sort: SiteSort | null = null, limit?: number) {
    return useUnlockedInfinite<SiteFilter, Site>({
        queryKey: keys.sites.list(filter, sort, limit),
        fetch: listSites,
        filters: filter,
        sort,
        limit,
    });
}

export function useSite(id: string | null) {
    return useUnlockedQuery({
        queryKey: keys.sites.detail(id ?? ""),
        queryFn: ({ signal }) => getSite({ id: id ?? "" }, signal),
        enabled: id !== null && id !== "",
    });
}

export function useCreateSite() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof createSite>[0]) => createSite(request),
        onSuccess: (answered) => {
            client.setQueryData(keys.sites.detail(answered.site.id), answered);
            void client.invalidateQueries({ queryKey: keys.sites.lists() });
        },
    });
}

export function useUpdateSite() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof updateSite>[0]) => updateSite(request),
        onSuccess: (answered) => {
            client.setQueryData(keys.sites.detail(answered.site.id), answered);
            void client.invalidateQueries({ queryKey: keys.sites.lists() });
        },
    });
}

export function useTestConnection() {
    return useMutation({
        mutationFn: (request: Parameters<typeof testConnection>[0]) => testConnection(request),
    });
}

export function useSiteDataHeld(siteId: string, enabled: boolean) {
    return useUnlockedQuery({
        queryKey: keys.sites.held(siteId),
        queryFn: ({ signal }) => siteDataCounts({ id: siteId }, signal),
        enabled: enabled && siteId !== "",
    });
}

export function useClearSiteData() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof clearSiteData>[0]) => clearSiteData(request),
        meta: quietMeta(["CONFLICT", "INVALID"]),
        onSuccess: () => {
            for (const root of [
                keys.sites.root(),
                keys.pages.root(),
                keys.findings.root(),
                keys.graph.root(),
                keys.runs.root(),
                keys.agent.root(),
                keys.reports.root(),
                keys.sync.root(),
            ]) {
                void client.invalidateQueries({ queryKey: root });
            }
        },
    });
}

export function useDeleteSite() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteSite>[0]) => deleteSite(request),
        meta: quietMeta(["CONFLICT"]),
        onSuccess: (_answered, request) => {
            client.removeQueries({ queryKey: keys.sites.detail(request.id) });
            void client.invalidateQueries({ queryKey: keys.sites.lists() });
        },
    });
}
