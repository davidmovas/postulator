import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useApplySheet, useInspectSheet, usePreviewSheet } from "../../data/hooks/imports.js";
import type { ImportMapping, ImportOptions, ImportSheet } from "../../data/types.js";
import type { ColumnMap } from "./columns.js";
import { forget, readRecent, remember, writeRecent } from "./recent.js";
import type { RecentFile } from "./recent.js";

export interface ImportFlow {
    inspect: ReturnType<typeof useInspectSheet>;
    preview: ReturnType<typeof usePreviewSheet>;
    apply: ReturnType<typeof useApplySheet>;
    columns: ColumnMap | null;
    detected: ColumnMap | null;
    options: ImportOptions;
    sheets: readonly ImportSheet[];
    headers: readonly string[];
    mapping: ImportMapping;
    recent: readonly RecentFile[];
    setColumns: (next: Record<string, string>) => void;
    setOptions: (next: ImportOptions) => void;
    useSaved: (saved: ImportMapping) => void;
    forgetFile: (path: string) => void;
    startApply: (saveMappingAs: string) => void;
}

function serialise(mapping: ImportMapping): string {
    return JSON.stringify([mapping.columns, mapping.options]);
}

export function useImportFlow(siteId: string, path: string, previewing: boolean): ImportFlow {
    const inspect = useInspectSheet();
    const preview = usePreviewSheet();
    const apply = useApplySheet();
    const [chosen, setChosen] = useState<Record<string, string> | null>(null);
    const [options, setOptions] = useState<ImportOptions>({});
    const [recent, setRecent] = useState<readonly RecentFile[]>(() => readRecent(siteId));
    const [sheets, setSheets] = useState<readonly ImportSheet[]>([]);
    const inspected = useRef("");
    const reading = useMemo(
        () => JSON.stringify([options.sheets ?? [], options.noHeader === true]),
        [options.sheets, options.noHeader],
    );
    const previewed = useRef("");
    const remembered = useRef("");

    const inspectMutate = inspect.mutate;
    const previewMutate = preview.mutate;
    const applyReset = apply.reset;

    useEffect(() => {
        setRecent(readRecent(siteId));
    }, [siteId]);

    useEffect(() => {
        if (siteId === "" || path === "") {
            return;
        }
        const stamp = `${path}|${reading}`;
        if (inspected.current === stamp) {
            return;
        }
        const opened = inspected.current.split("|")[0] !== path;
        inspected.current = stamp;
        previewed.current = "";
        setChosen(null);
        applyReset();
        if (opened) {
            setSheets([]);
            setOptions({});
            inspectMutate({ siteId, path, sheets: [], noHeader: false });
            return;
        }
        inspectMutate({ siteId, path, sheets: options.sheets ?? [], noHeader: options.noHeader === true });
    }, [siteId, path, reading, options.sheets, options.noHeader, inspectMutate, applyReset]);

    const detected = inspect.data?.detected.columns ?? null;

    const mapping = useMemo<ImportMapping>(
        () => ({ columns: chosen ?? detected, options, createdAt: null, updatedAt: null }),
        [chosen, detected, options],
    );

    useEffect(() => {
        const answered = inspect.data;
        if (answered === undefined || path === "" || remembered.current === path) {
            return;
        }
        remembered.current = path;
        setSheets(answered.sheets ?? []);
        const next = remember(readRecent(siteId), { path, rows: answered.rows, at: new Date().toISOString() });
        writeRecent(siteId, next);
        setRecent(next);
    }, [inspect.data, siteId, path]);

    useEffect(() => {
        if (!previewing || siteId === "" || path === "" || inspect.data === undefined) {
            return;
        }
        const stamp = `${path}|${serialise(mapping)}`;
        if (previewed.current === stamp) {
            return;
        }
        previewed.current = stamp;
        previewMutate({ siteId, path, mapping });
    }, [previewing, siteId, path, mapping, inspect.data, previewMutate]);

    const useSaved = useCallback((saved: ImportMapping) => {
        const next: Record<string, string> = {};
        for (const [field, column] of Object.entries(saved.columns ?? {})) {
            if (column !== undefined && column !== "") {
                next[field] = column;
            }
        }
        setChosen(next);
        setOptions({ ...saved.options });
    }, []);

    const forgetFile = useCallback(
        (dropped: string) => {
            const next = forget(readRecent(siteId), dropped);
            writeRecent(siteId, next);
            setRecent(next);
        },
        [siteId],
    );

    const applyMutate = apply.mutate;
    const startApply = useCallback(
        (saveMappingAs: string) => {
            applyMutate({ siteId, path, mapping, options: saveMappingAs === "" ? {} : { saveMappingAs } });
        },
        [applyMutate, siteId, path, mapping],
    );

    return {
        inspect,
        preview,
        apply,
        columns: mapping.columns,
        detected,
        options,
        sheets,
        headers: inspect.data?.headers ?? [],
        mapping,
        recent,
        setColumns: setChosen,
        setOptions,
        useSaved,
        forgetFile,
        startApply,
    };
}
