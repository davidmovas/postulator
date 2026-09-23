import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
    createPolicy,
    createTemplate,
    deleteOverride,
    deletePolicy,
    deleteTemplate,
    getEffectivePolicy,
    getPolicy,
    getTemplate,
    listPolicies,
    listTemplates,
    resolveForPage,
    setOverride,
    updatePolicy,
    updateTemplate,
} from "../endpoints/templates.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { PolicySort, TemplateSort } from "../sorts.js";
import type { LinkPolicy, PolicyFilter, Template, TemplateFilter } from "../types.js";

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

export function usePolicies(filter: PolicyFilter = {}, sort: PolicySort | null = null, limit?: number) {
    return useUnlockedInfinite<PolicyFilter, LinkPolicy>({
        queryKey: keys.policies.list(filter, sort, limit),
        fetch: listPolicies,
        filters: filter,
        sort,
        limit,
    });
}

export function usePolicy(id: string | null) {
    return useUnlockedQuery({
        queryKey: keys.policies.detail(id ?? ""),
        queryFn: ({ signal }) => getPolicy({ id: id ?? "" }, signal),
        enabled: id !== null && id !== "",
    });
}

export function useEffectivePolicy(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.policies.effective(siteId ?? ""),
        queryFn: ({ signal }) => getEffectivePolicy({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

function usePolicyWrite<Request, Answer extends { policy: LinkPolicy }>(
    call: (request: Request) => Promise<Answer>,
) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        onSuccess: (answered) => {
            client.setQueryData(keys.policies.detail(answered.policy.id), answered);
            void client.invalidateQueries({ queryKey: keys.policies.lists() });
            void client.invalidateQueries({ queryKey: keys.policies.effectives() });
        },
    });
}

export function useCreatePolicy() {
    return usePolicyWrite((request: Parameters<typeof createPolicy>[0]) => createPolicy(request));
}

export function useUpdatePolicy() {
    return usePolicyWrite((request: Parameters<typeof updatePolicy>[0]) => updatePolicy(request));
}

export function useDeletePolicy() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deletePolicy>[0]) => deletePolicy(request),
        onSuccess: (_answered, request) => {
            client.removeQueries({ queryKey: keys.policies.detail(request.id) });
            void client.invalidateQueries({ queryKey: keys.policies.lists() });
            void client.invalidateQueries({ queryKey: keys.policies.effectives() });
        },
    });
}
