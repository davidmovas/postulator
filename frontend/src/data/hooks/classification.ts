import { useMutation, useQueryClient } from "@tanstack/react-query";

import { quietMeta } from "../client.js";
import {
    applyClassification,
    deleteRule,
    listRules,
    previewClassification,
    previewClassificationPages,
    reorderRules,
    saveRule,
    testRule,
} from "../endpoints/templates.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { ClassificationPage, ClassificationPagesFilter, TestRuleRequest } from "../types.js";

export const previewPageSize = 50;

export function useRules(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.classification.rules(siteId ?? ""),
        queryFn: ({ signal }) => listRules({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

function useRuleWrite<Request, Answer>(call: (request: Request) => Promise<Answer>) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.classification.root() });
        },
    });
}

export function useSaveRule() {
    return useRuleWrite((request: Parameters<typeof saveRule>[0]) => saveRule(request));
}

export function useDeleteRule() {
    return useRuleWrite((request: Parameters<typeof deleteRule>[0]) => deleteRule(request));
}

export function useReorderRules() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof reorderRules>[0]) => reorderRules(request),
        onSuccess: (answered, request) => {
            client.setQueryData(keys.classification.rules(request.siteId), answered);
            void client.invalidateQueries({ queryKey: keys.classification.previews() });
        },
    });
}

export function useRuleTest(request: TestRuleRequest | null) {
    return useUnlockedQuery({
        queryKey: keys.classification.test(request),
        queryFn: ({ signal }) => testRule(request ?? { siteId: "", pageId: "" }, signal),
        enabled: request !== null && request.pageId !== "",
        quiet: ["INVALID"],
        refetchOnWindowFocus: false,
    });
}

export function useClassificationPreview(siteId: string, enabled: boolean) {
    return useUnlockedQuery({
        queryKey: keys.classification.preview(siteId),
        queryFn: ({ signal }) => previewClassification({ siteId }, signal),
        enabled: enabled && siteId !== "",
        staleTime: 0,
        refetchOnWindowFocus: false,
    });
}

export function useClassificationPages(filter: ClassificationPagesFilter, enabled: boolean) {
    return useUnlockedInfinite<ClassificationPagesFilter, ClassificationPage>({
        queryKey: keys.classification.pages(filter, previewPageSize),
        fetch: previewClassificationPages,
        filters: filter,
        limit: previewPageSize,
        enabled: enabled && filter.siteId !== "",
    });
}

export function useApplyClassification() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof applyClassification>[0]) => applyClassification(request),
        meta: quietMeta(["CONFLICT", "INVALID"]),
        onSettled: () => {
            void client.invalidateQueries({ queryKey: keys.classification.root() });
            void client.invalidateQueries({ queryKey: keys.pages.root() });
            void client.invalidateQueries({ queryKey: keys.sites.typesAll() });
            void client.invalidateQueries({ queryKey: keys.findings.root() });
            void client.invalidateQueries({ queryKey: keys.graph.fulls() });
        },
    });
}
