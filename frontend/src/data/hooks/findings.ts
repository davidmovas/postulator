import { useMutation, useQueryClient } from "@tanstack/react-query";

import { quietMeta } from "../client.js";
import { collectAll } from "../collect.js";
import {
    countFindings,
    explainResolution,
    getFinding,
    listFindings,
    resolveFinding,
    resolveFindings,
} from "../endpoints/findings.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { ExplainRequest, Finding, FindingCountsRequest, FindingFilter } from "../types.js";

export function useFindings(filter: FindingFilter, limit?: number) {
    return useUnlockedInfinite<FindingFilter, Finding>({
        queryKey: keys.findings.list(filter, limit),
        fetch: listFindings,
        filters: filter,
        limit,
        enabled: filter.siteId !== "",
    });
}

async function findingIds(filter: FindingFilter, signal?: AbortSignal): Promise<string[]> {
    return (await collectAll((args: Parameters<typeof listFindings>[0]) => listFindings(args, signal), filter)).map((held) => held.id);
}

export function useFindingIds(filter: FindingFilter, enabled: boolean) {
    return useUnlockedQuery({
        queryKey: keys.findings.ids(filter),
        queryFn: ({ signal }) => findingIds(filter, signal),
        enabled: enabled && filter.siteId !== "",
        staleTime: 0,
    });
}

export function useCollectFindingIds() {
    return useMutation({
        mutationFn: (filter: FindingFilter) => findingIds(filter),
    });
}

export function useFinding(id: string | null) {
    return useUnlockedQuery({
        queryKey: keys.findings.detail(id ?? ""),
        queryFn: ({ signal }) => getFinding({ findingId: id ?? "" }, signal),
        enabled: id !== null && id !== "",
        quiet: ["NOT_FOUND"],
    });
}

export function useFindingCounts(request: FindingCountsRequest) {
    return useUnlockedQuery({
        queryKey: keys.findings.counts(request),
        queryFn: ({ signal }) => countFindings(request, signal),
        enabled: request.siteId !== "",
    });
}

export function useExplanation(request: ExplainRequest | null) {
    return useUnlockedQuery({
        queryKey: keys.findings.explain(request ?? { findingIds: [], option: "" }),
        queryFn: ({ signal }) => explainResolution(request ?? { findingIds: [], option: "" }, signal),
        enabled: request !== null && (request.findingIds ?? []).length > 0 && request.option !== "",
        quiet: ["CONFLICT", "INVALID", "NOT_FOUND"],
    });
}

function useSettled() {
    const client = useQueryClient();
    return (): void => {
        void client.invalidateQueries({ queryKey: keys.findings.root() });
        void client.invalidateQueries({ queryKey: keys.pages.models() });
        void client.invalidateQueries({ queryKey: keys.pages.maps() });
    };
}

export function useResolveFinding() {
    const settled = useSettled();
    return useMutation({
        mutationFn: (request: Parameters<typeof resolveFinding>[0]) => resolveFinding(request),
        meta: quietMeta(["CONFLICT"]),
        onSuccess: settled,
    });
}

export function useResolveFindings() {
    const settled = useSettled();
    return useMutation({
        mutationFn: (request: Parameters<typeof resolveFindings>[0]) => resolveFindings(request),
        meta: quietMeta(["CONFLICT"]),
        onSuccess: settled,
    });
}
