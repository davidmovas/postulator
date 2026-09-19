import type { PageSort } from "../../data/sorts.js";
import type { PageFilter } from "../../data/types.js";
import { isOneOf, pageSortFields, pageStatuses } from "../../generated/vocab.js";

export type PagesView = "table" | "tree";

export interface PagesQuery {
    view: PagesView;
    status: string;
    entityId: string;
    unmapped: boolean;
    pathPrefix: string;
    sort: PageSort | null;
}

export const defaultQuery: PagesQuery = {
    view: "table",
    status: "",
    entityId: "",
    unmapped: false,
    pathPrefix: "",
    sort: null,
};

export function parseSort(raw: string | null): PageSort | null {
    if (raw === null || raw === "") {
        return null;
    }
    const separator = raw.lastIndexOf(":");
    if (separator <= 0) {
        return null;
    }
    const field = raw.slice(0, separator);
    const direction = raw.slice(separator + 1);
    if (!isOneOf(pageSortFields, field) || (direction !== "asc" && direction !== "desc")) {
        return null;
    }
    return { field, desc: direction === "desc" };
}

export function formatSort(sort: PageSort | null): string {
    return sort === null ? "" : `${sort.field}:${sort.desc ? "desc" : "asc"}`;
}

export function readQuery(params: URLSearchParams): PagesQuery {
    const status = params.get("status") ?? "";
    return {
        view: params.get("view") === "tree" ? "tree" : "table",
        status: isOneOf(pageStatuses, status) ? status : "",
        entityId: params.get("entity") ?? "",
        unmapped: params.get("unmapped") === "1",
        pathPrefix: params.get("prefix") ?? "",
        sort: parseSort(params.get("sort")),
    };
}

export function writeQuery(query: PagesQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (query.view !== defaultQuery.view) {
        params.set("view", query.view);
    }
    if (query.status !== "") {
        params.set("status", query.status);
    }
    if (query.entityId !== "") {
        params.set("entity", query.entityId);
    }
    if (query.unmapped) {
        params.set("unmapped", "1");
    }
    if (query.pathPrefix !== "") {
        params.set("prefix", query.pathPrefix);
    }
    const sort = formatSort(query.sort);
    if (sort !== "") {
        params.set("sort", sort);
    }
    return params;
}

export function searchOf(query: PagesQuery): string {
    const serialised = writeQuery(query).toString();
    return serialised === "" ? "" : `?${serialised}`;
}

export function filterOf(siteId: string, query: PagesQuery): PageFilter {
    const filter: PageFilter = { siteId };
    if (query.status !== "") {
        filter.status = query.status;
    }
    if (query.entityId !== "") {
        filter.entityId = query.entityId;
    }
    if (query.unmapped) {
        filter.unmapped = true;
    }
    if (query.pathPrefix !== "") {
        filter.pathPrefix = query.pathPrefix;
    }
    return filter;
}

export function narrowed(query: PagesQuery): boolean {
    return query.status !== "" || query.entityId !== "" || query.unmapped || query.pathPrefix !== "";
}

export function nextSort(current: PageSort | null, field: PageSort["field"]): PageSort | null {
    if (current === null || current.field !== field) {
        return { field, desc: false };
    }
    return current.desc ? null : { field, desc: true };
}
