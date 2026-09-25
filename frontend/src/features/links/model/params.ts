import { isOneOf, pageStatuses } from "../../../generated/vocab.js";

export const shows = [
    "all",
    "missing",
    "missingRequired",
    "blocked",
    "offGraph",
    "unpublished",
    "pending",
    "orphans",
    "skipped",
] as const;

export type Show = (typeof shows)[number];

export type LinksSort = "severity" | "path";

export interface LinksQuery {
    show: Show;
    entity: string;
    status: string;
    sort: LinksSort;
}

export const defaultQuery: LinksQuery = { show: "all", entity: "", status: "", sort: "severity" };

export function isShow(value: string): value is Show {
    return (shows as readonly string[]).includes(value);
}

export function readQuery(params: URLSearchParams): LinksQuery {
    const show = params.get("show") ?? "";
    const status = params.get("status") ?? "";
    return {
        show: isShow(show) ? show : "all",
        entity: params.get("entity") ?? "",
        status: isOneOf(pageStatuses, status) ? status : "",
        sort: params.get("sort") === "path" ? "path" : "severity",
    };
}

export function writeQuery(query: LinksQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (query.show !== defaultQuery.show) {
        params.set("show", query.show);
    }
    if (query.entity !== "") {
        params.set("entity", query.entity);
    }
    if (query.status !== "") {
        params.set("status", query.status);
    }
    if (query.sort !== defaultQuery.sort) {
        params.set("sort", query.sort);
    }
    return params;
}

export function searchOf(query: LinksQuery): string {
    const serialised = writeQuery(query).toString();
    return serialised === "" ? "" : `?${serialised}`;
}

export function narrowed(query: LinksQuery): boolean {
    return query.show !== "all" || query.entity !== "" || query.status !== "";
}
