import type { TemplateSort } from "../../data/sorts.js";
import { isOneOf, templateSortFields } from "../../generated/vocab.js";

export type TemplatesTab = "templates" | "policies";

export type ScopeFilter = "all" | "global" | "site";

const scopeFilters: readonly ScopeFilter[] = ["all", "global", "site"];

export type SortChoice = "nameAsc" | "nameDesc" | "newest" | "oldest";

export interface TemplatesQuery {
    scope: ScopeFilter;
    pageKind: string;
    search: string;
    sort: TemplateSort | null;
}

export const defaultQuery: TemplatesQuery = { scope: "all", pageKind: "", search: "", sort: null };

export const sorts: Readonly<Record<SortChoice, TemplateSort>> = {
    nameAsc: { field: "name", desc: false },
    nameDesc: { field: "name", desc: true },
    newest: { field: "createdAt", desc: true },
    oldest: { field: "createdAt", desc: false },
};

export function sortChoice(sort: TemplateSort | null): SortChoice {
    if (sort === null) {
        return "nameAsc";
    }
    if (sort.field === "createdAt") {
        return sort.desc ? "newest" : "oldest";
    }
    return sort.desc ? "nameDesc" : "nameAsc";
}

export function parseSort(raw: string | null): TemplateSort | null {
    if (raw === null || raw === "") {
        return null;
    }
    const separator = raw.lastIndexOf(":");
    if (separator <= 0) {
        return null;
    }
    const field = raw.slice(0, separator);
    const direction = raw.slice(separator + 1);
    if (!isOneOf(templateSortFields, field) || (direction !== "asc" && direction !== "desc")) {
        return null;
    }
    return { field, desc: direction === "desc" };
}

export function formatSort(sort: TemplateSort | null): string {
    return sort === null ? "" : `${sort.field}:${sort.desc ? "desc" : "asc"}`;
}

export function readQuery(params: URLSearchParams): TemplatesQuery {
    const scope = params.get("scope") ?? "";
    return {
        scope: isOneOf(scopeFilters, scope) ? scope : defaultQuery.scope,
        pageKind: params.get("kind") ?? "",
        search: params.get("q") ?? "",
        sort: parseSort(params.get("sort")),
    };
}

export function writeQuery(query: TemplatesQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (query.scope !== defaultQuery.scope) {
        params.set("scope", query.scope);
    }
    if (query.pageKind !== "") {
        params.set("kind", query.pageKind);
    }
    if (query.search !== "") {
        params.set("q", query.search);
    }
    const sort = formatSort(query.sort);
    if (sort !== "") {
        params.set("sort", sort);
    }
    return params;
}

export function searchOf(query: TemplatesQuery): string {
    const serialised = writeQuery(query).toString();
    return serialised === "" ? "" : `?${serialised}`;
}

export function narrowed(query: TemplatesQuery): boolean {
    return query.scope !== defaultQuery.scope || query.pageKind !== "" || query.search !== "";
}

export const actionParam = "action";
export const actionNew = "new";

export function wantsNew(params: URLSearchParams): boolean {
    return params.get(actionParam) === actionNew;
}
