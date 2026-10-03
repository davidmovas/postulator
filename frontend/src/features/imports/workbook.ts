import type { ImportMapping, ImportOptions, ImportSheet, ImportSheetMapping } from "../../data/types.js";
import type { ImportField } from "../../generated/vocab.js";
import type { ColumnMap } from "./columns.js";
import { assign, usable } from "./columns.js";

export interface SheetSettings {
    columns: ColumnMap;
    options: ImportOptions;
    mappingId: string;
}

export interface Workbook {
    sheets: readonly ImportSheet[];
    chosen: readonly string[];
    active: string;
    perSheet: Readonly<Record<string, SheetSettings>>;
}

export interface WorkbookRequest {
    mapping: ImportMapping;
    sheets?: ImportSheetMapping[];
}

export interface SheetsRead {
    mapping: { options: { sheets?: readonly string[] | null } };
    sheets?: readonly { sheet: string }[] | null;
}

export type ColumnsNotice =
    | { kind: "ready" }
    | { kind: "noSheet" }
    | { kind: "unmatched" }
    | { kind: "needsTarget" }
    | { kind: "otherSheet"; sheet: string };

const blank: SheetSettings = { columns: {}, options: {}, mappingId: "" };

function filled(columns: ColumnMap | null | undefined): Record<string, string> {
    const kept: Record<string, string> = {};
    for (const [field, column] of Object.entries(columns ?? {})) {
        if (column !== undefined && column !== "") {
            kept[field] = column;
        }
    }
    return kept;
}

function named(name: string): string[] {
    return name === "" ? [] : [name];
}

function namesOf(book: Workbook): string[] {
    return book.sheets.map((sheet) => sheet.name);
}

export function sheetOf(book: Workbook, name: string): ImportSheet | undefined {
    return book.sheets.find((sheet) => sheet.name === name);
}

function detectedOf(sheet: ImportSheet): SheetSettings {
    return { columns: filled(sheet.detected.columns), options: { ...sheet.detected.options }, mappingId: "" };
}

export function openWorkbook(sheets: readonly ImportSheet[]): Workbook {
    const withRows = sheets.filter((sheet) => sheet.rows > 0).map((sheet) => sheet.name);
    const chosen = withRows.length > 0 ? withRows : sheets.slice(0, 1).map((sheet) => sheet.name);
    const perSheet: Record<string, SheetSettings> = {};
    for (const sheet of sheets) {
        perSheet[sheet.name] = detectedOf(sheet);
    }
    return { sheets, chosen, active: chosen[0] ?? "", perSheet };
}

export function settingsOf(book: Workbook, name: string = book.active): SheetSettings {
    return book.perSheet[name] ?? blank;
}

function withSettings(book: Workbook, name: string, settings: SheetSettings): Workbook {
    return { ...book, perSheet: { ...book.perSheet, [name]: settings } };
}

export function chooseSheet(book: Workbook, name: string, on: boolean): Workbook {
    if (sheetOf(book, name) === undefined) {
        return book;
    }
    const held = new Set(book.chosen);
    if (on) {
        held.add(name);
    } else {
        held.delete(name);
    }
    const chosen = namesOf(book).filter((each) => held.has(each));
    const active = on ? name : held.has(book.active) ? book.active : (chosen[0] ?? "");
    return { ...book, chosen, active };
}

export function activateSheet(book: Workbook, name: string): Workbook {
    return book.chosen.includes(name) ? { ...book, active: name } : book;
}

export function assignColumn(book: Workbook, header: string, field: ImportField | null): Workbook {
    const held = settingsOf(book);
    return withSettings(book, book.active, { ...held, columns: assign(held.columns, header, field) });
}

export function setSheetOptions(book: Workbook, options: ImportOptions): Workbook {
    return withSettings(book, book.active, { ...settingsOf(book), options });
}

export function headerless(book: Workbook, on: boolean): Workbook {
    const sheet = sheetOf(book, book.active);
    if (sheet === undefined) {
        return book;
    }
    if (!on) {
        return withSettings(book, book.active, detectedOf(sheet));
    }
    const held = settingsOf(book);
    return withSettings(book, book.active, {
        columns: {},
        options: { ...held.options, noHeader: true, indentColumns: [], levelColumns: [], noteColumns: [] },
        mappingId: "",
    });
}

export function adoptSavedOn(book: Workbook, saved: ImportMapping, name: string): Workbook {
    if (sheetOf(book, name) === undefined) {
        return book;
    }
    return withSettings(book, name, {
        columns: filled(saved.columns),
        options: { ...saved.options, sheets: named(name) },
        mappingId: saved.id ?? "",
    });
}

export function adoptSaved(book: Workbook, saved: ImportMapping): Workbook {
    const present = new Set(namesOf(book));
    const listed = (saved.options.sheets ?? []).filter((name) => present.has(name));
    const targets = listed.length > 0 ? listed : [book.active === "" ? (namesOf(book)[0] ?? "") : book.active];
    let next = book;
    for (const name of targets) {
        next = adoptSavedOn(next, saved, name);
        if (!next.chosen.includes(name)) {
            next = chooseSheet(next, name, true);
        }
    }
    return activateSheet(next, targets[0] ?? next.active);
}

function readable(settings: SheetSettings): boolean {
    return usable(settings.columns, settings.options.indentColumns ?? [], settings.options.levelColumns ?? []);
}

function unreadable(book: Workbook): string[] {
    return book.chosen.filter((name) => !readable(settingsOf(book, name)));
}

export function columnsNotice(book: Workbook): ColumnsNotice {
    if (book.chosen.length === 0) {
        return { kind: "noSheet" };
    }
    const held = settingsOf(book);
    if (!readable(held)) {
        const sheet = sheetOf(book, book.active);
        const detected = sheet === undefined ? blank : detectedOf(sheet);
        const matched = usable(detected.columns, [], detected.options.levelColumns ?? []);
        return { kind: held.options.noHeader !== true && !matched ? "unmatched" : "needsTarget" };
    }
    const other = unreadable(book)[0];
    return other === undefined ? { kind: "ready" } : { kind: "otherSheet", sheet: other };
}

export function chosenRows(book: Workbook): number {
    return rowsOf(book.sheets.filter((sheet) => book.chosen.includes(sheet.name)));
}

export function rowsOf(sheets: readonly ImportSheet[]): number {
    return sheets.reduce((total, sheet) => total + sheet.rows, 0);
}

export function inUse(book: Workbook): string[] {
    const ids = book.chosen.map((name) => settingsOf(book, name).mappingId).filter((id) => id !== "");
    return [...new Set(ids)];
}

function mappingOf(book: Workbook, name: string): ImportMapping {
    const held = settingsOf(book, name);
    return {
        columns: filled(held.columns),
        options: { ...held.options, sheets: named(name) },
        createdAt: null,
        updatedAt: null,
    };
}

export function requestOf(book: Workbook): WorkbookRequest | null {
    const [first, ...rest] = book.chosen;
    if (first === undefined) {
        return null;
    }
    if (rest.length === 0) {
        return { mapping: mappingOf(book, first) };
    }
    return {
        mapping: { columns: null, options: {}, createdAt: null, updatedAt: null },
        sheets: book.chosen.map((name) => ({ sheet: name, mapping: mappingOf(book, name) })),
    };
}

export function sheetsOf(request: SheetsRead | null | undefined): string[] {
    if (request === null || request === undefined) {
        return [];
    }
    const each = request.sheets ?? [];
    if (each.length > 0) {
        return each.map((held) => held.sheet);
    }
    return [...(request.mapping.options.sheets ?? [])];
}

export function savedNames(name: string, request: SheetsRead | null | undefined): string[] {
    const trimmed = name.trim();
    if (trimmed === "" || request === null || request === undefined) {
        return [];
    }
    const each = request.sheets ?? [];
    return each.length === 0 ? [trimmed] : each.map((held) => `${trimmed} / ${held.sheet}`);
}
