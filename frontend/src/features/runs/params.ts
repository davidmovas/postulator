import { choiceParam, queryCodec, sortParam } from "../../data/params.js";
import type { RunSort } from "../../data/sorts.js";
import type { RunFilter } from "../../data/types.js";
import { itemStatuses, runKinds, runSortFields, runStatuses } from "../../generated/vocab.js";

export interface RunsQuery {
    status: string;
    kind: string;
    sort: RunSort | null;
}

const codec = queryCodec<RunsQuery>({
    status: choiceParam("status", runStatuses, ""),
    kind: choiceParam("kind", runKinds, ""),
    sort: sortParam("sort", runSortFields),
});

export const defaultQuery: RunsQuery = codec.defaults;

export const readQuery = codec.read;

export const writeQuery = codec.write;

export const searchOf = codec.search;

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
    return codec.carries(query, ["status", "kind"]);
}

interface ItemQuery {
    status: string;
}

const itemCodec = queryCodec<ItemQuery>({
    status: choiceParam("item", itemStatuses, ""),
});

export function readItemStatus(params: URLSearchParams): string {
    return itemCodec.read(params).status;
}

export function itemSearchOf(status: string): string {
    return itemCodec.search({ status });
}
