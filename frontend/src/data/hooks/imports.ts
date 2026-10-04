import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
    applySheet,
    deleteMapping,
    exportSite,
    inspectSheet,
    listMappings,
    previewSheet,
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

interface SheetSampleRequest {
    siteId: string;
    path: string;
    sheet: string;
    noHeader: boolean;
}

export function useSheetSample(request: SheetSampleRequest, enabled: boolean) {
    const { siteId, path, sheet, noHeader } = request;
    return useUnlockedQuery({
        queryKey: [...keys.imports.inspect(siteId, path), sheet, noHeader],
        queryFn: ({ signal }) => inspectSheet({ siteId, path, sheets: sheet === "" ? [] : [sheet], noHeader }, signal),
        enabled: enabled && siteId !== "" && path !== "",
        refetchOnWindowFocus: false,
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

export function useDeleteMapping() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteMapping>[0]) => deleteMapping(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.imports.mappingsAll() });
        },
    });
}
