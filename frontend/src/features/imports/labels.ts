import { copy } from "../../copy/index.js";
import {
    blockingImportFindingCodes,
    cannibalizationReasons,
    edgeKinds,
    exportFormats,
    importActions,
    importColumnUses,
    importFields,
    importFindingCodes,
    importRowTypes,
    isOneOf,
} from "../../generated/vocab.js";
import type { ImportField } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";

export function fieldLabel(field: string): string {
    return isOneOf(importFields, field) ? copy.imports.fields[field] : field;
}

export interface ColumnFate {
    header: string;
    use: string;
    field?: string;
}

export function columnUseLabel(column: ColumnFate): string {
    if (column.use === "field") {
        return fieldLabel(column.field ?? "");
    }
    return isOneOf(importColumnUses, column.use) ? copy.imports.columnUses[column.use] : column.use;
}

export function findingLabel(code: string): string {
    return isOneOf(importFindingCodes, code) ? copy.imports.findings[code] : code;
}

export function actionLabel(action: string): string {
    return isOneOf(importActions, action) ? copy.imports.actions[action] : action;
}

export function reasonLabel(reason: string): string {
    return isOneOf(cannibalizationReasons, reason) ? copy.imports.reasons[reason] : reason;
}

export function edgeKindLabel(kind: string): string {
    return isOneOf(edgeKinds, kind) ? copy.imports.edgeKinds[kind] : kind;
}

export function exportFormatLabel(format: string): string {
    return isOneOf(exportFormats, format) ? copy.imports.export.formats[format] : format;
}

export function rowTypeLabel(rowType: string): string {
    return isOneOf(importRowTypes, rowType) ? copy.imports.rowTypes[rowType] : rowType;
}

export interface ProductRow {
    wpType: string;
    plannedPath?: string;
    storeName?: string;
    matchedBy?: string;
}

const matchedBy = copy.imports.preview.matchedBy as Readonly<Record<string, string>>;

export function productNote(page: ProductRow): string | null {
    if (page.wpType !== "product") {
        return null;
    }
    const name = page.storeName ?? "";
    if (name === "") {
        return copy.imports.preview.waitingProduct;
    }
    const by = matchedBy[page.matchedBy ?? ""] ?? matchedBy["path"] ?? "";
    return copy.imports.preview.storeProduct(name, by, page.plannedPath ?? "");
}

export function blocking(code: string): boolean {
    return (blockingImportFindingCodes as readonly string[]).includes(code);
}

export function actionTone(action: string): Tone {
    switch (action) {
        case "create":
            return "ok";
        case "update":
            return "info";
        default:
            return "muted";
    }
}

export const noField = "none";

export interface FieldChoice {
    value: ImportField | typeof noField;
    label: string;
}

export const fieldChoices: readonly FieldChoice[] = [
    { value: noField, label: copy.imports.fields.none },
    ...importFields.map((field) => ({ value: field, label: copy.imports.fields[field] })),
];
