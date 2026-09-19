import type { RunSort } from "../../data/sorts.js";
import type { RunFilter } from "../../data/types.js";
import { isOneOf, itemStatuses, runKinds, runSortFields, runStatuses } from "../../generated/vocab.js";

export interface RunsQuery {
    status: string;
    kind: string;
    sort: RunSort | null;
}

export const defaultQuery: RunsQuery = { status: "", kind: "", sort: null };

export function parseSort(raw: string | null): RunSort | null {
    if (raw === null || raw === "") {
        return null;
    }
    const separator = raw.lastIndexOf(":");
    if (separator <= 0) {
        return null;
    }
    const field = raw.slice(0, separator);
    const direction = raw.slice(separator + 1);
    if (!isOneOf(runSortFields, field) || (direction !== "asc" && direction !== "desc")) {
        return null;
    }
    return { field, desc: direction === "desc" };
}

export function formatSort(sort: RunSort | null): string {
    return sort === null ? "" : `${sort.field}:${sort.desc ? "desc" : "asc"}`;
}

export function readQuery(params: URLSearchParams): RunsQuery {
    const status = params.get("status") ?? "";
    const kind = params.get("kind") ?? "";
    return {
        status: isOneOf(runStatuses, status) ? status : "",
        kind: isOneOf(runKinds, kind) ? kind : "",
        sort: parseSort(params.get("sort")),
    };
}

export function writeQuery(query: RunsQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (query.status !== "") {
        params.set("status", query.status);
    }
    if (query.kind !== "") {
        params.set("kind", query.kind);
    }
    const sort = formatSort(query.sort);
    if (sort !== "") {
        params.set("sort", sort);
    }
    return params;
}

export function searchOf(query: RunsQuery): string {
    const serialised = writeQuery(query).toString();
    return serialised === "" ? "" : `?${serialised}`;
}

export function filterOf(siteId: string, query: RunsQuery): RunFilter {
    const filter: RunFilter = { siteId };
    if (query.status !== "") {
        filter.status = query.status;
    }
    if (query.kind !== "") {
        filter.kind = query.kind;
    }
    return filter;
}

export function narrowed(query: RunsQuery): boolean {
    return query.status !== "" || query.kind !== "";
}

export function nextSort(current: RunSort | null, field: RunSort["field"]): RunSort | null {
    if (current === null || current.field !== field) {
        return { field, desc: false };
    }
    return current.desc ? null : { field, desc: true };
}

export function readItemStatus(params: URLSearchParams): string {
    const status = params.get("item") ?? "";
    return isOneOf(itemStatuses, status) ? status : "";
}

export function itemSearchOf(status: string): string {
    return status === "" ? "" : `?item=${status}`;
}
