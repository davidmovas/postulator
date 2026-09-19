import type { TemplateSort } from "../../data/sorts.js";
import { isOneOf, templateSortFields } from "../../generated/vocab.js";

export type TemplatesTab = "templates" | "policies";

export interface TemplatesQuery {
    tab: TemplatesTab;
    sort: TemplateSort | null;
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
    return {
        tab: params.get("tab") === "policies" ? "policies" : "templates",
        sort: parseSort(params.get("sort")),
    };
}

export function writeQuery(query: TemplatesQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (query.tab !== "templates") {
        params.set("tab", query.tab);
    }
    const sort = formatSort(query.sort);
    if (sort !== "") {
        params.set("sort", sort);
    }
    return params;
}
