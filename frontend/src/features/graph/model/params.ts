import { entityKinds, isOneOf } from "../../../generated/vocab.js";
import type { Lens } from "./lens.js";
import { isLens } from "./lens.js";

export type GraphView = "map" | "outline";

export interface GraphQuery {
    view: GraphView;
    lens: Lens;
    kinds: readonly string[];
    isolate: boolean;
}

export const defaultQuery: GraphQuery = { view: "map", lens: "all", kinds: [], isolate: false };

export function readQuery(params: URLSearchParams): GraphQuery {
    const lens = params.get("lens") ?? "";
    const kinds = (params.get("kinds") ?? "")
        .split(",")
        .map((kind) => kind.trim())
        .filter((kind) => isOneOf(entityKinds, kind));
    return {
        view: params.get("view") === "outline" ? "outline" : "map",
        lens: isLens(lens) ? lens : "all",
        kinds,
        isolate: params.get("isolate") === "1",
    };
}

export function writeQuery(query: GraphQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (query.view !== defaultQuery.view) {
        params.set("view", query.view);
    }
    if (query.lens !== defaultQuery.lens) {
        params.set("lens", query.lens);
    }
    if (query.kinds.length > 0) {
        params.set("kinds", query.kinds.join(","));
    }
    if (query.isolate) {
        params.set("isolate", "1");
    }
    return params;
}

export function searchOf(query: GraphQuery): string {
    const serialised = writeQuery(query).toString();
    return serialised === "" ? "" : `?${serialised}`;
}
