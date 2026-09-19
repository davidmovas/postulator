import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
    createPage,
    deletePage,
    getPage,
    listPages,
    mapPageToEntity,
    pageTree,
    replacePageLinks,
    setCanonicalPage,
    unmapPage,
    updatePage,
} from "../endpoints/pages.js";
import { quietMeta } from "../client.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { PageSort } from "../sorts.js";
import type { Page, PageFilter } from "../types.js";

export function usePages(filter: PageFilter, sort: PageSort | null = null, limit?: number) {
    return useUnlockedInfinite<PageFilter, Page>({
        queryKey: keys.pages.list(filter, sort, limit),
        fetch: listPages,
        filters: filter,
        sort,
        limit,
        enabled: filter.siteId !== "",
    });
}

export function usePage(id: string | null) {
    return useUnlockedQuery({
        queryKey: keys.pages.detail(id ?? ""),
        queryFn: ({ signal }) => getPage({ id: id ?? "" }, signal),
        enabled: id !== null && id !== "",
    });
}

export function usePageTree(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.pages.tree(siteId ?? ""),
        queryFn: ({ signal }) => pageTree({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

function usePageWrite<Request, Answer extends { page: Page }>(
    call: (request: Request) => Promise<Answer>,
) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        meta: quietMeta(["CONFLICT"]),
        onSuccess: (answered) => {
            void client.invalidateQueries({ queryKey: keys.pages.detail(answered.page.id) });
            void client.invalidateQueries({ queryKey: keys.pages.lists() });
            void client.invalidateQueries({ queryKey: keys.pages.tree(answered.page.siteId) });
            void client.invalidateQueries({ queryKey: keys.graph.entityAll() });
        },
    });
}

export function useCreatePage() {
    return usePageWrite((request: Parameters<typeof createPage>[0]) => createPage(request));
}

export function useUpdatePage() {
    return usePageWrite((request: Parameters<typeof updatePage>[0]) => updatePage(request));
}

export function useMapPageToEntity() {
    return usePageWrite((request: Parameters<typeof mapPageToEntity>[0]) => mapPageToEntity(request));
}

export function useUnmapPage() {
    return usePageWrite((request: Parameters<typeof unmapPage>[0]) => unmapPage(request));
}

export function useSetCanonicalPage() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof setCanonicalPage>[0]) => setCanonicalPage(request),
        onSuccess: (answered) => {
            void client.invalidateQueries({ queryKey: keys.pages.detail(answered.page.id) });
            void client.invalidateQueries({ queryKey: keys.pages.lists() });
            void client.invalidateQueries({ queryKey: keys.graph.root() });
        },
    });
}

export function useReplacePageLinks() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof replacePageLinks>[0]) => replacePageLinks(request),
        onSuccess: (_answered, request) => {
            void client.invalidateQueries({ queryKey: keys.pages.detail(request.pageId) });
        },
    });
}

export function useDeletePage() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deletePage>[0]) => deletePage(request),
        onSuccess: (_answered, request) => {
            client.removeQueries({ queryKey: keys.pages.detail(request.id) });
            void client.invalidateQueries({ queryKey: keys.pages.lists() });
            void client.invalidateQueries({ queryKey: keys.pages.trees() });
        },
    });
}
