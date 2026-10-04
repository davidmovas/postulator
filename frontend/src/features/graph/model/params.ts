import { choiceParam, flagParam, listParam, queryCodec } from "../../../data/params.js";
import { entityKinds } from "../../../generated/vocab.js";
import { nodeStates } from "./index.js";
import type { NodeState } from "./index.js";
import type { Lens } from "./lens.js";
import { lenses } from "./lens.js";

const graphViews = ["map", "outline"] as const;

export type GraphView = (typeof graphViews)[number];

export interface GraphQuery {
    view: GraphView;
    lens: Lens;
    kinds: readonly string[];
    states: readonly NodeState[];
    isolate: boolean;
    proof: boolean;
}

const codec = queryCodec<GraphQuery>({
    view: choiceParam("view", graphViews, "map"),
    lens: choiceParam("lens", lenses, "all"),
    kinds: listParam("kinds", entityKinds),
    states: listParam("states", nodeStates),
    isolate: flagParam("isolate"),
    proof: flagParam("proof"),
});

export const defaultQuery: GraphQuery = codec.defaults;

export const readQuery = codec.read;

export const writeQuery = codec.write;

export const searchOf = codec.search;
