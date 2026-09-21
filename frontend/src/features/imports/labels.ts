import { copy } from "../../copy/index.js";
import {
    blockingImportFindingCodes,
    cannibalizationReasons,
    edgeKinds,
    entityKinds,
    importActions,
    importFields,
    importFindingCodes,
    isOneOf,
} from "../../generated/vocab.js";
import type { ImportField } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";

export function fieldLabel(field: string): string {
    return isOneOf(importFields, field) ? copy.imports.fields[field] : field;
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

export function entityKindLabel(kind: string): string {
    return isOneOf(entityKinds, kind) ? copy.imports.entityKinds[kind] : kind;
}

export function edgeKindLabel(kind: string): string {
    return isOneOf(edgeKinds, kind) ? copy.imports.edgeKinds[kind] : kind;
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
