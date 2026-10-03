import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useApplySheet, useInspectSheet, usePreviewSheet, useSheetSample } from "../../data/hooks/imports.js";
import type { ImportMapping, ImportOptions } from "../../data/types.js";
import type { ImportField } from "../../generated/vocab.js";
import type { ColumnMap } from "./columns.js";
import { forget, readRecent, remember, writeRecent } from "./recent.js";
import type { RecentFile } from "./recent.js";
import type { Workbook, WorkbookRequest } from "./workbook.js";
import {
    activateSheet,
    adoptSaved,
    adoptSavedOn,
    assignColumn,
    chooseSheet,
    headerless,
    openWorkbook,
    requestOf,
    rowsOf,
    setSheetOptions,
    settingsOf,
    sheetOf,
} from "./workbook.js";

interface SheetReading {
    headers: readonly string[];
    sample: readonly (readonly string[] | null)[];
    busy: boolean;
    failure: unknown;
}

export interface ImportFlow {
    inspect: ReturnType<typeof useInspectSheet>;
    preview: ReturnType<typeof usePreviewSheet>;
    apply: ReturnType<typeof useApplySheet>;
    book: Workbook | null;
    reading: SheetReading;
    detected: ColumnMap | null;
    request: WorkbookRequest | null;
    saveAs: string;
    recent: readonly RecentFile[];
    choose: (sheet: string, on: boolean) => void;
    activate: (sheet: string) => void;
    assign: (header: string, field: ImportField | null) => void;
    setOptions: (next: ImportOptions) => void;
    setHeaderless: (on: boolean) => void;
    adopt: (saved: ImportMapping) => void;
    adoptHere: (saved: ImportMapping) => void;
    setSaveAs: (name: string) => void;
    forgetFile: (path: string) => void;
    startApply: () => void;
}

export function useImportFlow(siteId: string, path: string, previewing: boolean): ImportFlow {
    const inspect = useInspectSheet();
    const preview = usePreviewSheet();
    const apply = useApplySheet();
    const [book, setBook] = useState<Workbook | null>(null);
    const [pending, setPending] = useState<ImportMapping | null>(null);
    const [saveAs, setSaveAs] = useState("");
    const [recent, setRecent] = useState<readonly RecentFile[]>(() => readRecent(siteId));
    const opened = useRef("");
    const adopted = useRef<unknown>(null);
    const previewed = useRef("");

    const inspectMutate = inspect.mutate;
    const previewMutate = preview.mutate;
    const previewReset = preview.reset;
    const applyMutate = apply.mutate;
    const applyReset = apply.reset;

    useEffect(() => {
        setRecent(readRecent(siteId));
    }, [siteId]);

    useEffect(() => {
        const stamp = siteId === "" || path === "" ? "" : `${siteId}|${path}`;
        if (opened.current === stamp) {
            return;
        }
        opened.current = stamp;
        previewed.current = "";
        setBook(null);
        setSaveAs("");
        applyReset();
        previewReset();
        if (stamp !== "") {
            inspectMutate({ siteId, path, sheets: [], noHeader: false });
        }
    }, [siteId, path, inspectMutate, applyReset, previewReset]);

    useEffect(() => {
        const answered = inspect.data;
        if (answered === undefined || adopted.current === answered || opened.current === "") {
            return;
        }
        adopted.current = answered;
        const sheets = answered.sheets ?? [];
        setBook(openWorkbook(sheets));
        const rows = sheets.length > 1 ? rowsOf(sheets) : answered.rows;
        const next = remember(readRecent(siteId), { path, rows, at: new Date().toISOString() });
        writeRecent(siteId, next);
        setRecent(next);
    }, [inspect.data, siteId, path]);

    useEffect(() => {
        if (book === null || pending === null) {
            return;
        }
        setBook(adoptSaved(book, pending));
        setPending(null);
    }, [book, pending]);

    const first = inspect.data;
    const active = book?.active ?? "";
    const settings = book === null ? null : settingsOf(book);
    const noHeader = settings?.options.noHeader === true;
    const sheet = book === null ? undefined : sheetOf(book, active);
    const fromFirst = first !== undefined && !noHeader && sheet !== undefined && first.sheets?.[0]?.name === active;
    const sampled = useSheetSample(
        { siteId, path, sheet: active, noHeader },
        book !== null && sheet !== undefined && book.chosen.length > 0 && !fromFirst,
    );
    const answered = fromFirst ? first : sampled.data;

    const reading = useMemo<SheetReading>(
        () => ({
            headers: noHeader ? (answered?.headers ?? []) : (sheet?.headers ?? answered?.headers ?? []),
            sample: answered?.sample ?? [],
            busy: !fromFirst && sampled.isLoading,
            failure: fromFirst ? null : sampled.error,
        }),
        [noHeader, answered, sheet, fromFirst, sampled.isLoading, sampled.error],
    );

    const detected = noHeader || sheet === undefined ? null : (sheet.detected.columns ?? null);
    const request = useMemo(() => (book === null ? null : requestOf(book)), [book]);

    useEffect(() => {
        if (!previewing || siteId === "" || path === "" || request === null) {
            return;
        }
        const stamp = `${path}|${JSON.stringify(request)}`;
        if (previewed.current === stamp) {
            return;
        }
        previewed.current = stamp;
        previewMutate({ siteId, path, ...request });
    }, [previewing, siteId, path, request, previewMutate]);

    const edit = useCallback((change: (held: Workbook) => Workbook) => {
        setBook((held) => (held === null ? held : change(held)));
    }, []);

    const choose = useCallback(
        (name: string, on: boolean) => {
            edit((held) => chooseSheet(held, name, on));
        },
        [edit],
    );

    const activate = useCallback(
        (name: string) => {
            edit((held) => activateSheet(held, name));
        },
        [edit],
    );

    const assign = useCallback(
        (header: string, field: ImportField | null) => {
            edit((held) => assignColumn(held, header, field));
        },
        [edit],
    );

    const setOptions = useCallback(
        (next: ImportOptions) => {
            edit((held) => setSheetOptions(held, next));
        },
        [edit],
    );

    const setHeaderless = useCallback(
        (on: boolean) => {
            edit((held) => headerless(held, on));
        },
        [edit],
    );

    const adopt = useCallback(
        (saved: ImportMapping) => {
            if (book === null) {
                setPending(saved);
                return;
            }
            edit((held) => adoptSaved(held, saved));
        },
        [book, edit],
    );

    const adoptHere = useCallback(
        (saved: ImportMapping) => {
            edit((held) => adoptSavedOn(held, saved, held.active));
        },
        [edit],
    );

    const forgetFile = useCallback(
        (dropped: string) => {
            const next = forget(readRecent(siteId), dropped);
            writeRecent(siteId, next);
            setRecent(next);
        },
        [siteId],
    );

    const startApply = useCallback(() => {
        if (request === null) {
            return;
        }
        const name = saveAs.trim();
        applyMutate({ siteId, path, ...request, options: name === "" ? {} : { saveMappingAs: name } });
    }, [applyMutate, siteId, path, request, saveAs]);

    return {
        inspect,
        preview,
        apply,
        book,
        reading,
        detected,
        request,
        saveAs,
        recent,
        choose,
        activate,
        assign,
        setOptions,
        setHeaderless,
        adopt,
        adoptHere,
        setSaveAs,
        forgetFile,
        startApply,
    };
}
