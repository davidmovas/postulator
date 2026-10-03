import { keepPreviousData, useMutation, useQueryClient } from "@tanstack/react-query";

import {
    disableModel,
    getProfiles,
    listCalls,
    listModels,
    setProfile,
    spendReport,
    testProvider,
    upsertModel,
    usageSummary,
} from "../endpoints/models.js";
import type { UsageScope } from "../keys.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { ModelCall, ModelCallFilter } from "../types.js";

export function useModelCatalog() {
    return useUnlockedQuery({
        queryKey: keys.models.catalog(),
        queryFn: ({ signal }) => listModels({}, signal),
    });
}

export function useRoleProfiles(siteId?: string) {
    return useUnlockedQuery({
        queryKey: keys.models.profiles(siteId),
        queryFn: ({ signal }) => getProfiles({ siteId }, signal),
    });
}

export function useUsage(usageScope: UsageScope = {}) {
    return useUnlockedQuery({
        queryKey: keys.models.usage(usageScope),
        queryFn: ({ signal }) => usageSummary(usageScope, signal),
    });
}

export function useSpendReport(days: number) {
    return useUnlockedQuery({
        queryKey: keys.models.spendOver(days),
        queryFn: ({ signal }) => spendReport({ days }, signal),
        placeholderData: keepPreviousData,
    });
}

export function useRunSpend(runId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.models.spendOfRun(runId ?? ""),
        queryFn: ({ signal }) => spendReport({ runId: runId ?? "" }, signal),
        enabled: runId !== null && runId !== "",
    });
}

export function useModelCalls(filter: ModelCallFilter = {}, limit?: number) {
    return useUnlockedInfinite<ModelCallFilter, ModelCall>({
        queryKey: keys.models.calls(filter, limit),
        fetch: listCalls,
        filters: filter,
        sort: null,
        limit,
    });
}

export function useUpsertModel() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof upsertModel>[0]) => upsertModel(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.models.catalog() });
        },
    });
}

export function useDisableModel() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof disableModel>[0]) => disableModel(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.models.catalog() });
            void client.invalidateQueries({ queryKey: keys.models.profilesAll() });
        },
    });
}

export function useSetProfile() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof setProfile>[0]) => setProfile(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.models.profilesAll() });
        },
    });
}

export function useTestProvider() {
    return useMutation({
        mutationFn: (request: Parameters<typeof testProvider>[0]) => testProvider(request),
    });
}
