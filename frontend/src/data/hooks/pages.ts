import { keepPreviousData, useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import {
    createPage,
    deletePage,
    getPage,
    listPages,
    mapPageToEntity,
    modelSummary,
    pageModel,
    pageTree,
    previewLink,
    setCanonicalPage,
    setPageTemplates,
    setPageType,
    siteMap,
    unmapPage,
    updatePage,
    writerPack,
} from "../endpoints/pages.js";
import { infiniteGcTimeMs, quietMeta } from "../client.js";
import { keys } from "../keys.js";
import { useLockGate } from "../lock.js";
import { useUnlockedInfinite, useUnlockedQueries, useUnlockedQuery } from "../query.js";
import type { PageSort } from "../sorts.js";
import type { Page, PageFilter, SiteMapFilter, SiteMapView } from "../types.js";
import type { Cursor } from "../../lib/paging.js";

export const siteMapLimit = 100;

export function useSiteMap(filter: SiteMapFilter, enabled = true) {
    const gate = useLockGate();
    return useInfiniteQuery({
        queryKey: keys.pages.map(filter, siteMapLimit),
        enabled: enabled && filter.siteId !== "" && gate.ready && !gate.locked,
        initialPageParam: undefined as Cursor | undefined,
        queryFn: ({ pageParam, signal }) =>
            siteMap({ ...filter, limit: siteMapLimit, ...(pageParam === undefined ? {} : { cursor: pageParam }) }, signal),
        getNextPageParam: (last: SiteMapView) =>
            last.nodes.hasMore && last.nodes.nextCursor !== undefined ? last.nodes.nextCursor : undefined,
        getPreviousPageParam: () => undefined,
        refetchOnWindowFocus: false,
        gcTime: infiniteGcTimeMs,
        placeholderData: keepPreviousData,
    });
}

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

export function usePagePaths(ids: readonly string[], enabled: boolean): ReadonlyMap<string, string> {
    const answered = useUnlockedQueries(
        ids.map((id) => ({
            queryKey: keys.pages.detail(id),
            queryFn: ({ signal }: { signal: AbortSignal }) => getPage({ id }, signal),
            enabled: enabled && id !== "",
        })),
    );
    const out = new Map<string, string>();
    answered.forEach((held, at) => {
        const id = ids[at];
        const path = held.data?.page.path;
        if (id !== undefined && path !== undefined) {
            out.set(id, path);
        }
    });
    return out;
}

export function usePageTree(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.pages.tree(siteId ?? ""),
        queryFn: ({ signal }) => pageTree({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

export function usePageModel(pageId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.pages.model(pageId ?? ""),
        queryFn: ({ signal }) => pageModel({ pageId: pageId ?? "" }, signal),
        enabled: pageId !== null && pageId !== "",
    });
}

export function useModelSummary(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.pages.summary(siteId ?? ""),
        queryFn: ({ signal }) => modelSummary({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

export function useWriterPack(pageId: string | null, field: string) {
    return useUnlockedQuery({
        queryKey: keys.pages.pack(pageId ?? "", field),
        queryFn: ({ signal }) => writerPack({ pageId: pageId ?? "", field, budget: 0 }, signal),
        enabled: pageId !== null && pageId !== "",
        quiet: ["INVALID"],
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

export function useSetPageType() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof setPageType>[0]) => setPageType(request),
        onSuccess: (answered) => {
            void client.invalidateQueries({ queryKey: keys.pages.detail(answered.page.id) });
            void client.invalidateQueries({ queryKey: keys.pages.lists() });
            void client.invalidateQueries({ queryKey: keys.pages.tree(answered.page.siteId) });
            void client.invalidateQueries({ queryKey: keys.sites.types(answered.page.siteId) });
            void client.invalidateQueries({ queryKey: keys.templates.resolved(answered.page.id) });
            void client.invalidateQueries({ queryKey: keys.graph.full(answered.page.siteId) });
        },
    });
}

export function useSetPageTemplates() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof setPageTemplates>[0]) => setPageTemplates(request),
        meta: quietMeta(["CONFLICT"]),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.pages.root() });
            void client.invalidateQueries({ queryKey: keys.sites.typesAll() });
            void client.invalidateQueries({ queryKey: keys.templates.resolvedAll() });
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

export const previewLinkStaleMs = 50 * 60_000;

export function usePreviewLink(page: Pick<Page, "id" | "status"> | null) {
    return useUnlockedQuery({
        queryKey: keys.previews.link(page?.id ?? "", page?.status ?? ""),
        queryFn: ({ signal }) => previewLink({ pageId: page?.id ?? "" }, signal),
        enabled: page !== null && page.id !== "" && page.status !== "planned" && page.status !== "archived",
        staleTime: previewLinkStaleMs,
        gcTime: previewLinkStaleMs,
        refetchOnWindowFocus: false,
        refetchOnMount: false,
    });
}
