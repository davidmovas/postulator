import type { Sort } from "../lib/paging.js";

export const siteSortFields = ["createdAt", "name"] as const;
export const pageSortFields = ["createdAt", "path"] as const;
export const runSortFields = ["createdAt", "status"] as const;
export const templateSortFields = ["createdAt", "name"] as const;
export const policySortFields = ["createdAt", "name"] as const;
export const entitySortFields = ["createdAt", "name"] as const;

export type SortOf<Fields extends readonly string[]> = { field: Fields[number]; desc: boolean };

export type SiteSort = SortOf<typeof siteSortFields>;
export type PageSort = SortOf<typeof pageSortFields>;
export type RunSort = SortOf<typeof runSortFields>;
export type TemplateSort = SortOf<typeof templateSortFields>;
export type PolicySort = SortOf<typeof policySortFields>;
export type EntitySort = SortOf<typeof entitySortFields>;

export type NoSort = null;

export function sortSegment(sort: Sort | null | undefined): string {
    if (sort === null || sort === undefined) {
        return "default";
    }
    return `${sort.field}:${sort.desc ? "desc" : "asc"}`;
}
