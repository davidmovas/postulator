import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
    createPageType,
    createTemplate,
    deleteOverride,
    deletePageType,
    deleteTemplate,
    getTemplate,
    listPageTypes,
    listTemplates,
    resolveForPage,
    savePageType,
    setOverride,
    setScope,
    updateTemplate,
} from "../endpoints/templates.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { TemplateSort } from "../sorts.js";
import type { Template, TemplateFilter } from "../types.js";

export function useTemplates(
    filter: TemplateFilter = {},
    sort: TemplateSort | null = null,
    limit?: number,
) {
    return useUnlockedInfinite<TemplateFilter, Template>({
        queryKey: keys.templates.list(filter, sort, limit),
        fetch: listTemplates,
        filters: filter,
        sort,
        limit,
    });
}

export function useTemplate(id: string | null) {
    return useUnlockedQuery({
        queryKey: keys.templates.detail(id ?? ""),
        queryFn: ({ signal }) => getTemplate({ id: id ?? "" }, signal),
        enabled: id !== null && id !== "",
    });
}

export function useResolvedTemplate(pageId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.templates.resolved(pageId ?? ""),
        queryFn: ({ signal }) => resolveForPage({ pageId: pageId ?? "" }, signal),
        enabled: pageId !== null && pageId !== "",
    });
}

function useTemplateWrite<Request, Answer extends { template: Template }>(
    call: (request: Request) => Promise<Answer>,
) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        onSuccess: (answered) => {
            void client.invalidateQueries({ queryKey: keys.templates.detail(answered.template.id) });
            void client.invalidateQueries({ queryKey: keys.templates.lists() });
            void client.invalidateQueries({ queryKey: keys.templates.resolvedAll() });
        },
    });
}

export function useCreateTemplate() {
    return useTemplateWrite((request: Parameters<typeof createTemplate>[0]) => createTemplate(request));
}

export function useUpdateTemplate() {
    return useTemplateWrite((request: Parameters<typeof updateTemplate>[0]) => updateTemplate(request));
}

export function useDeleteTemplate() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteTemplate>[0]) => deleteTemplate(request),
        onSuccess: (_answered, request) => {
            client.removeQueries({ queryKey: keys.templates.detail(request.id) });
            void client.invalidateQueries({ queryKey: keys.templates.lists() });
            void client.invalidateQueries({ queryKey: keys.templates.resolvedAll() });
        },
    });
}

export function useSetOverride() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof setOverride>[0]) => setOverride(request),
        onSuccess: (answered) => {
            void client.invalidateQueries({ queryKey: keys.templates.detail(answered.override.templateId) });
            void client.invalidateQueries({ queryKey: keys.templates.resolvedAll() });
        },
    });
}

export function useDeleteOverride() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteOverride>[0]) => deleteOverride(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.templates.details() });
            void client.invalidateQueries({ queryKey: keys.templates.resolvedAll() });
        },
    });
}

export function usePageTypes(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.sites.types(siteId ?? ""),
        queryFn: ({ signal }) => listPageTypes({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

function usePageTypeWrite<Request, Answer>(call: (request: Request) => Promise<Answer>, movesPages: boolean) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.sites.typesAll() });
            if (movesPages) {
                void client.invalidateQueries({ queryKey: keys.pages.root() });
                void client.invalidateQueries({ queryKey: keys.graph.fulls() });
            }
        },
    });
}

export function useCreatePageType() {
    return usePageTypeWrite((request: Parameters<typeof createPageType>[0]) => createPageType(request), false);
}

export function useSavePageType() {
    return usePageTypeWrite((request: Parameters<typeof savePageType>[0]) => savePageType(request), false);
}

export function useDeletePageType() {
    return usePageTypeWrite((request: Parameters<typeof deletePageType>[0]) => deletePageType(request), true);
}

export function useSetScope() {
    return usePageTypeWrite((request: Parameters<typeof setScope>[0]) => setScope(request), false);
}
