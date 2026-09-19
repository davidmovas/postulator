import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
    createSite,
    deleteSite,
    getSite,
    listSites,
    updateSite,
} from "../endpoints/sites.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { SiteSort } from "../sorts.js";
import type { Site, SiteFilter } from "../types.js";

export function useSites(filter: SiteFilter = {}, sort: SiteSort | null = null, limit?: number) {
    return useUnlockedInfinite<SiteFilter, Site>({
        queryKey: keys.sites.list(filter, sort),
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

export function useDeleteSite() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteSite>[0]) => deleteSite(request),
        onSuccess: (_answered, request) => {
            client.removeQueries({ queryKey: keys.sites.detail(request.id) });
            void client.invalidateQueries({ queryKey: keys.sites.lists() });
        },
    });
}
