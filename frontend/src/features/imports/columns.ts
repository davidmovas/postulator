import { importFields } from "../../generated/vocab.js";
import type { ImportField } from "../../generated/vocab.js";

export type ColumnMap = Readonly<Record<string, string | undefined>>;

export function targetOf(columns: ColumnMap | null, header: string): ImportField | null {
    if (columns === null || header === "") {
        return null;
    }
    for (const field of importFields) {
        if (columns[field] === header) {
            return field;
        }
    }
    return null;
}

export function assign(columns: ColumnMap | null, header: string, field: ImportField | null): Record<string, string> {
    const next: Record<string, string> = {};
    for (const known of importFields) {
        const held = columns?.[known];
        if (held !== undefined && held !== "" && held !== header) {
            next[known] = held;
        }
    }
    if (field !== null) {
        next[field] = header;
    }
    return next;
}

export function mappedFields(columns: ColumnMap | null): ImportField[] {
    return importFields.filter((field) => {
        const held = columns?.[field];
        return held !== undefined && held !== "";
    });
}

export function unmappedHeaders(headers: readonly string[], columns: ColumnMap | null): string[] {
    return headers.filter((header) => header !== "" && targetOf(columns, header) === null);
}

export function usable(
    columns: ColumnMap | null,
    indentColumns: readonly string[] = [],
    levelColumns: readonly string[] = [],
): boolean {
    if (indentColumns.length > 0 || levelColumns.length > 0) {
        return true;
    }
    const mapped = mappedFields(columns);
    return mapped.includes("path") || mapped.includes("entity");
}

export function freeHeaders(headers: readonly string[], columns: ColumnMap | null, elsewhere: readonly string[]): string[] {
    return unmappedHeaders(headers, columns).filter((header) => !elsewhere.includes(header));
}

export function groupHeaders(
    headers: readonly string[],
    columns: ColumnMap | null,
    elsewhere: readonly string[],
    roots: readonly string[],
): string[] {
    return freeHeaders(headers, columns, elsewhere).filter((header) => roots.includes(header));
}

export function toggled(chosen: readonly string[], header: string, headers: readonly string[]): string[] {
    const held = new Set(chosen);
    if (held.has(header)) {
        held.delete(header);
    } else {
        held.add(header);
    }
    return headers.filter((each) => held.has(each));
}

export function takenFrom(columns: ColumnMap | null, detected: ColumnMap | null, header: string): boolean {
    const target = targetOf(columns, header);
    return target !== null && targetOf(detected, header) === target;
}
