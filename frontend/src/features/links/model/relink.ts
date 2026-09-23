import type { PageAudit } from "../../../data/types.js";

export const relinkCap = 500;

export interface RelinkSelection {
    pageIds: readonly string[];
    total: number;
    capped: boolean;
}

export function relinkSelection(rows: readonly PageAudit[]): RelinkSelection {
    const owed = rows.filter((row) => row.missing > 0 && row.skipReason === "");
    return {
        pageIds: owed.slice(0, relinkCap).map((row) => row.pageId),
        total: owed.length,
        capped: owed.length > relinkCap,
    };
}
