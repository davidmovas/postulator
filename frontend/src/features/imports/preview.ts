import type { PreviewReport } from "../../data/types.js";

export type PreviewColumn = NonNullable<PreviewReport["columns"]>[number];

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
