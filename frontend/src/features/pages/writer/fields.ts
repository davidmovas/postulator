import type { PageType, WriterPackSection } from "../../../data/types.js";
import { fieldLabel } from "../../settings/types/labels.js";
import { linkField, orderedScopes } from "../../settings/types/model.js";

export interface WriterField {
    key: string;
    label: string;
}

const writtenBesideTheLinks: ReadonlySet<string> = new Set([
    "excerpt",
    "seo_title",
    "seo_description",
    "product_short_description",
]);

export function writerFields(type: PageType | null): WriterField[] {
    const main = linkField(type?.role ?? "");
    const scopes = orderedScopes(type?.scopes ?? []);
    const fields: WriterField[] = [
        { key: main, label: fieldLabel(main, scopes.find((scope) => scope.field === main)?.label ?? "") },
    ];
    for (const scope of scopes) {
        if (scope.enabled && scope.field !== main && writtenBesideTheLinks.has(scope.field)) {
            fields.push({ key: scope.field, label: fieldLabel(scope.field, scope.label) });
        }
    }
    return fields;
}

export function askedField(fields: readonly WriterField[], choice: string): string {
    const main = fields[0]?.key ?? "";
    return choice === main || !fields.some((field) => field.key === choice) ? "" : choice;
}

export function byPriority(sections: readonly WriterPackSection[]): WriterPackSection[] {
    return [...sections].sort((left, right) => left.priority - right.priority);
}

export interface BudgetRefusal {
    budget: number;
    needed: number;
}

export function budgetRefusal(details: Record<string, unknown> | undefined): BudgetRefusal | null {
    if (details?.["reason"] !== "budget_too_small") {
        return null;
    }
    const budget = details["budget"];
    const needed = details["needed"];
    return {
        budget: typeof budget === "number" ? budget : 0,
        needed: typeof needed === "number" ? needed : 0,
    };
}
