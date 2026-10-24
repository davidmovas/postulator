import { queryCodec, textParam } from "../../../data/params.js";
import type { SiteMapFilter } from "../../../data/types.js";

export interface MapQuery {
    under: string;
    search: string;
    type: string;
}

const codec = queryCodec<MapQuery>({
    under: textParam("under"),
    search: textParam("q"),
    type: textParam("type"),
});

export const defaultQuery: MapQuery = codec.defaults;

export const readQuery = codec.read;

export const writeQuery = codec.write;

export const searchOf = codec.search;

export function flat(query: MapQuery): boolean {
    return query.search.trim() !== "" || query.type !== "";
}

export function filterOf(siteId: string, query: MapQuery): SiteMapFilter {
    const filter: SiteMapFilter = { siteId };
    if (query.under !== "") {
        filter.parentId = query.under;
    }
    const search = query.search.trim();
    if (search !== "") {
        filter.search = search;
    }
    if (query.type !== "") {
        filter.pageTypeId = query.type;
    }
    return filter;
}
