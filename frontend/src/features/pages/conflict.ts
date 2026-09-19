import { failure } from "../../data/errors.js";

export const cannibalizationReasons = [
    "path_conflict",
    "same_entity_canonical",
    "same_primary_keyword",
] as const;

export type CannibalizationReason = (typeof cannibalizationReasons)[number];

export interface Offender {
    pageId: string;
    path: string;
    reason: string;
    entityId: string;
}

export type PageConflict =
    | { kind: "cannibalization"; offenders: readonly Offender[] }
    | { kind: "descendants"; descendants: number };

function text(held: unknown): string {
    return typeof held === "string" ? held : "";
}

function offenderOf(held: unknown): Offender | null {
    if (typeof held !== "object" || held === null) {
        return null;
    }
    const row = held as Record<string, unknown>;
    const pageId = text(row["pageId"]);
    const path = text(row["path"]);
    if (pageId === "" && path === "") {
        return null;
    }
    return { pageId, path, reason: text(row["reason"]), entityId: text(row["entityId"]) };
}

function offendersOf(held: unknown): readonly Offender[] | null {
    if (!Array.isArray(held)) {
        return null;
    }
    const rows: Offender[] = [];
    for (const entry of held) {
        const offender = offenderOf(entry);
        if (offender !== null) {
            rows.push(offender);
        }
    }
    return rows.length === 0 ? null : rows;
}

export function conflictOf(thrown: unknown): PageConflict | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reported = failure(thrown);
    if (reported.code !== "CONFLICT") {
        return null;
    }
    const details = reported.details ?? {};
    const offenders = offendersOf(details["evidence"]);
    if (offenders !== null) {
        return { kind: "cannibalization", offenders };
    }
    const descendants = details["descendants"];
    if (typeof descendants === "number" && descendants > 0) {
        return { kind: "descendants", descendants };
    }
    return null;
}
