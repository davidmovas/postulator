import { choiceParam, flagParam, queryCodec, sortParam, textParam } from "../../data/params.js";
import type { PageSort } from "../../data/sorts.js";
import type { PageFilter } from "../../data/types.js";
import { pageSortFields, pageStatuses } from "../../generated/vocab.js";

const pagesViews = ["table", "tree"] as const;

export type PagesView = (typeof pagesViews)[number];

export interface PagesQuery {
    view: PagesView;
    status: string;
    entityId: string;
    descendants: boolean;
    unmapped: boolean;
    pathPrefix: string;
    categoryId: string;
    sort: PageSort | null;
}

const codec = queryCodec<PagesQuery>({
    view: choiceParam("view", pagesViews, "table"),
    status: choiceParam("status", pageStatuses, ""),
    entityId: textParam("entity"),
    descendants: flagParam("under"),
    unmapped: flagParam("unmapped"),
    pathPrefix: textParam("prefix"),
    categoryId: textParam("category"),
    sort: sortParam("sort", pageSortFields),
});

export const defaultQuery: PagesQuery = codec.defaults;

export const readQuery = codec.read;

export const writeQuery = codec.write;

export const searchOf = codec.search;

export function filterOf(siteId: string, query: PagesQuery): PageFilter {
    const filter: PageFilter = { siteId };
    if (query.status !== "") {
        filter.status = query.status;
    }
    if (query.entityId !== "") {
        filter.entityId = query.entityId;
        if (query.descendants) {
            filter.includeDescendants = true;
        }
    }
    if (query.unmapped) {
        filter.unmapped = true;
    }
    if (query.pathPrefix !== "") {
        filter.pathPrefix = query.pathPrefix;
    }
    if (query.categoryId !== "") {
        filter.categoryId = query.categoryId;
    }
    return filter;
}

export function narrowed(query: PagesQuery): boolean {
    return codec.carries(query, ["status", "entityId", "unmapped", "pathPrefix", "categoryId"]);
}

const pageTabs = ["details", "links", "mapping", "report", "preview"] as const;

export type PageTab = (typeof pageTabs)[number];

const tabParam = choiceParam("tab", pageTabs, "details");

export function readTab(params: URLSearchParams): PageTab {
    return tabParam.read(params.get(tabParam.key));
}

export function withTab(params: URLSearchParams, tab: PageTab): URLSearchParams {
    const next = new URLSearchParams(params);
    next.delete(tabParam.key);
    const written = tabParam.write(tab);
    if (written !== "") {
        next.set(tabParam.key, written);
    }
    return next;
}
