import type { Sort } from "../lib/paging.js";
import {
    entitySortFields,
    isOneOf,
    pageSortFields,
    runSortFields,
    siteSortFields,
    templateSortFields,
} from "../generated/vocab.js";

export { entitySortFields, pageSortFields, runSortFields, siteSortFields, templateSortFields };

export const policySortFields = templateSortFields;

export type SortOf<Fields extends readonly string[]> = { field: Fields[number]; desc: boolean };

export type SiteSort = SortOf<typeof siteSortFields>;
export type PageSort = SortOf<typeof pageSortFields>;
export type RunSort = SortOf<typeof runSortFields>;
export type TemplateSort = SortOf<typeof templateSortFields>;
export type PolicySort = SortOf<typeof policySortFields>;
export type EntitySort = SortOf<typeof entitySortFields>;

export type NoSort = null;

export function parseSort<F extends string>(fields: readonly F[], raw: string | null): SortOf<readonly F[]> | null {
    if (raw === null || raw === "") {
        return null;
    }
    const separator = raw.lastIndexOf(":");
    if (separator <= 0) {
        return null;
    }
    const field = raw.slice(0, separator);
    const direction = raw.slice(separator + 1);
    if (!isOneOf(fields, field) || (direction !== "asc" && direction !== "desc")) {
        return null;
    }
    return { field, desc: direction === "desc" };
}

export function formatSort(sort: Sort | null | undefined): string {
    if (sort === null || sort === undefined) {
        return "";
    }
    return `${sort.field}:${sort.desc ? "desc" : "asc"}`;
}

export function sortSegment(sort: Sort | null | undefined): string {
    const formatted = formatSort(sort);
    return formatted === "" ? "default" : formatted;
}

export function cycleSort<F extends string>(
    current: SortOf<readonly F[]> | null,
    field: F,
): SortOf<readonly F[]> | null {
    if (current === null || current.field !== field) {
        return { field, desc: false };
    }
    return current.desc ? null : { field, desc: true };
}
