import type { VisibleRow } from "./fold.js";

export function pruneRows(rows: readonly VisibleRow[], kept: ReadonlySet<string> | null): VisibleRow[] {
    if (kept === null) {
        return [...rows];
    }
    const out: VisibleRow[] = [];
    let skippingBelow = -1;
    for (const row of rows) {
        if (skippingBelow >= 0 && row.depth > skippingBelow) {
            continue;
        }
        skippingBelow = -1;
        if (row.kind === "more") {
            if (kept.has(row.parentId)) {
                out.push(row);
            }
            continue;
        }
        if (kept.has(row.id)) {
            out.push(row);
        } else {
            skippingBelow = row.depth;
        }
    }
    return out;
}
