import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
    disableModel,
    getProfiles,
    listModels,
    setProfile,
    testProvider,
    upsertModel,
    usageSummary,
} from "../endpoints/models.js";
import type { UsageScope } from "../keys.js";
import { keys } from "../keys.js";
import { useUnlockedQuery } from "../query.js";

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
