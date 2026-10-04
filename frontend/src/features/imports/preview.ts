import type { PreviewReport } from "../../data/types.js";
import { categoryPathKey } from "../../domain/categories.js";
import type { ImportCategoryAction } from "../../generated/vocab.js";

export type PreviewColumn = NonNullable<PreviewReport["columns"]>[number];

export type PreviewCategory = NonNullable<PreviewReport["categories"]>[number];

const createAction: ImportCategoryAction = "create";

export function createdPaths(report: Pick<PreviewReport, "categories">): ReadonlySet<string> {
    const out = new Set<string>();
    for (const category of report.categories ?? []) {
        if (category.action === createAction) {
            out.add(categoryPathKey(category.path ?? []));
        }
    }
    return out;
}

interface SheetColumns {
    sheet: string;
    columns: PreviewColumn[];
}

interface FromSheet {
    sheet?: string;
}

export function sheetsIn(report: PreviewReport): string[] {
    const seen: string[] = [];
    const rows: readonly (readonly FromSheet[] | null)[] = [
        report.columns,
        report.pages,
        report.entities,
        report.groups,
        report.categories,
        report.edges,
    ];
    for (const listed of rows) {
        for (const row of listed ?? []) {
            const sheet = row.sheet ?? "";
            if (sheet !== "" && !seen.includes(sheet)) {
                seen.push(sheet);
            }
        }
    }
    return seen;
}

export function columnsBySheet(columns: readonly PreviewColumn[]): SheetColumns[] {
    const grouped: SheetColumns[] = [];
    for (const column of columns) {
        const sheet = column.sheet ?? "";
        const last = grouped[grouped.length - 1];
        if (last !== undefined && last.sheet === sheet) {
            last.columns.push(column);
        } else {
            grouped.push({ sheet, columns: [column] });
        }
    }
    return grouped;
}
