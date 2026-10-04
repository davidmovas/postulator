import { choiceParam, queryCodec, sortParam, textParam } from "../../data/params.js";
import type { TemplateSort } from "../../data/sorts.js";
import { templateSortFields } from "../../generated/vocab.js";

export type TemplatesTab = "templates" | "policies";

const scopeFilters = ["all", "global", "site"] as const;

export type ScopeFilter = (typeof scopeFilters)[number];

export type SortChoice = "nameAsc" | "nameDesc" | "newest" | "oldest";

export interface TemplatesQuery {
    scope: ScopeFilter;
    pageKind: string;
    search: string;
    sort: TemplateSort | null;
}

const codec = queryCodec<TemplatesQuery>({
    scope: choiceParam("scope", scopeFilters, "all"),
    pageKind: textParam("kind"),
    search: textParam("q"),
    sort: sortParam("sort", templateSortFields),
});

export const defaultQuery: TemplatesQuery = codec.defaults;

export const readQuery = codec.read;

export const writeQuery = codec.write;

export const searchOf = codec.search;

export function narrowed(query: TemplatesQuery): boolean {
    return codec.carries(query, ["scope", "pageKind", "search"]);
}

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
