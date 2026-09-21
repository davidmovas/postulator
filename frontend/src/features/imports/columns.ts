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

export function usable(columns: ColumnMap | null): boolean {
    const mapped = mappedFields(columns);
    return mapped.includes("path") || mapped.includes("entity");
}

export function takenFrom(columns: ColumnMap | null, detected: ColumnMap | null, header: string): boolean {
    const target = targetOf(columns, header);
    return target !== null && targetOf(detected, header) === target;
}
