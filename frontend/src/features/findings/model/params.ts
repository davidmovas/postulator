import { choiceParam, queryCodec, textParam } from "../../../data/params.js";
import type { FindingCountsRequest, FindingFilter } from "../../../data/types.js";
import { findingCodes, findingStatuses } from "../../../generated/vocab.js";
import type { FindingStatus } from "../../../generated/vocab.js";

export interface FindingsQuery {
    status: FindingStatus;
    code: string;
    type: string;
    under: string;
}

const codec = queryCodec<FindingsQuery>({
    status: choiceParam("status", findingStatuses, "open"),
    code: choiceParam("code", findingCodes, ""),
    type: textParam("type"),
    under: textParam("under"),
});

export const defaultQuery: FindingsQuery = codec.defaults;

export const readQuery = codec.read;

export const writeQuery = codec.write;

export const searchOf = codec.search;

export function narrowed(query: FindingsQuery): boolean {
    return codec.carries(query, ["code", "type", "under"]);
}

export function countsOf(siteId: string, query: FindingsQuery): FindingCountsRequest {
    const request: FindingCountsRequest = { siteId, status: query.status };
    if (query.type !== "") {
        request.pageTypeId = query.type;
    }
    if (query.under !== "") {
        request.under = query.under;
    }
    return request;
}

export function filterOf(siteId: string, query: FindingsQuery): FindingFilter {
    const filter: FindingFilter = { siteId, status: query.status };
    if (query.code !== "") {
        filter.code = query.code;
    }
    if (query.type !== "") {
        filter.pageTypeId = query.type;
    }
    if (query.under !== "") {
        filter.under = query.under;
    }
    return filter;
}

export function findingsAddress(siteId: string, under: string): string {
    return `/s/${siteId}/findings${searchOf({ ...defaultQuery, under })}`;
}
