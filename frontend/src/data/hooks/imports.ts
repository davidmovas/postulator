import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
    applySheet,
    deleteMapping,
    exportSite,
    inspectSheet,
    listMappings,
    previewSheet,
    saveMapping,
} from "../endpoints/imports.js";
import { keys } from "../keys.js";
import { useUnlockedQuery } from "../query.js";

export function useMappings(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.imports.mappings(siteId ?? ""),
        queryFn: ({ signal }) => listMappings({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

export function useInspectSheet() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof inspectSheet>[0]) => inspectSheet(request),
        onSuccess: (answered, request) => {
            client.setQueryData(keys.imports.inspect(request.siteId, request.path), answered);
        },
    });
}

export function usePreviewSheet() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof previewSheet>[0]) => previewSheet(request),
        onSuccess: (answered, request) => {
            client.setQueryData(
                keys.imports.preview(request.siteId, request.path, request.mapping.id ?? "detected"),
                answered,
            );
        },
    });
}

export function useApplySheet() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof applySheet>[0]) => applySheet(request),
        onSuccess: (_answered, request) => {
            void client.invalidateQueries({ queryKey: keys.pages.root() });
            void client.invalidateQueries({ queryKey: keys.graph.root() });
            void client.invalidateQueries({ queryKey: keys.imports.mappings(request.siteId) });
            void client.invalidateQueries({ queryKey: keys.reports.site(request.siteId) });
        },
    });
}

export function useExportSite() {
    return useMutation({
        mutationFn: (request: Parameters<typeof exportSite>[0]) => exportSite(request),
    });
}

export function useSaveMapping() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof saveMapping>[0]) => saveMapping(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.imports.mappingsAll() });
        },
    });
}

export function useDeleteMapping() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteMapping>[0]) => deleteMapping(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.imports.mappingsAll() });
        },
    });
}
