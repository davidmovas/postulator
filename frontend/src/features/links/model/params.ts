import { choiceParam, queryCodec, textParam } from "../../../data/params.js";
import { pageStatuses } from "../../../generated/vocab.js";

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

const linksSorts = ["severity", "path"] as const;

export type LinksSort = (typeof linksSorts)[number];

export interface LinksQuery {
    show: Show;
    entity: string;
    status: string;
    sort: LinksSort;
}

const codec = queryCodec<LinksQuery>({
    show: choiceParam("show", shows, "all"),
    entity: textParam("entity"),
    status: choiceParam("status", pageStatuses, ""),
    sort: choiceParam("sort", linksSorts, "severity"),
});

export const defaultQuery: LinksQuery = codec.defaults;

export const readQuery = codec.read;

export const writeQuery = codec.write;

export const searchOf = codec.search;

export function narrowed(query: LinksQuery): boolean {
    return codec.carries(query, ["show", "entity", "status"]);
}
