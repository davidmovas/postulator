import { copy } from "../../copy/index.js";
import {
    blockingImportFindingCodes,
    cannibalizationReasons,
    edgeKinds,
    exportFormats,
    importActions,
    importCategoryActions,
    importColumnUses,
    importFields,
    importFindingCodes,
    importRowTypes,
    isOneOf,
} from "../../generated/vocab.js";
import type { ImportCategoryAction, ImportField } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";

export function fieldLabel(field: string): string {
    return isOneOf(importFields, field) ? copy.imports.fields[field] : field;
}

export interface ColumnFate {
    header: string;
    use: string;
    field?: string;
}

const rootHeaders: readonly string[] = ["rootentity", "root"];

export function isRootLevel(header: string): boolean {
    return rootHeaders.includes(header.toLowerCase().replace(/[^\p{L}\p{N}]+/gu, ""));
}

export function levelUseLabel(header: string): string {
    return isRootLevel(header) ? copy.imports.rootLevel : copy.imports.columnUses.level;
}

export function columnUseLabel(column: ColumnFate): string {
    if (column.use === "field") {
        return fieldLabel(column.field ?? "");
    }
    if (column.use === "level") {
        return levelUseLabel(column.header);
    }
    return isOneOf(importColumnUses, column.use) ? copy.imports.columnUses[column.use] : column.use;
}

export function categoryActionLabel(action: string): string {
    return isOneOf(importCategoryActions, action) ? copy.imports.categoryActions[action] : action;
}

const categoryActionTones: Readonly<Record<ImportCategoryAction, Tone>> = {
    create: "ok",
    match: "muted",
    delete: "warn",
};

export function categoryActionTone(action: string): Tone {
    return isOneOf(importCategoryActions, action) ? categoryActionTones[action] : "muted";
}

export function findingLabel(code: string): string {
    return isOneOf(importFindingCodes, code) ? copy.imports.findings[code] : code;
}

interface FindingSpot {
    sheet?: string;
    row: number;
}

export function findingPlace(finding: FindingSpot): string | null {
    const sheet = finding.sheet ?? "";
    if (sheet !== "") {
        return copy.imports.preview.place(sheet, finding.row);
    }
    return finding.row > 0 ? copy.imports.preview.row(finding.row) : null;
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
