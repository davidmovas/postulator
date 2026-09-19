import { Graph } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { Edge, Entity } from "../types.js";

export const loadGraph = one(Graph.LoadGraph);
export const createEntity = one(Graph.CreateEntity);
export const updateEntity = one(Graph.UpdateEntity);
export const deleteEntity = one(Graph.DeleteEntity);
export const getEntity = one(Graph.GetEntity);
export const listEntities = listed<Parameters<typeof Graph.ListEntities>[0], Entity>(Graph.ListEntities);
export const setAnchors = one(Graph.SetAnchors);
export const addEdge = one(Graph.AddEdge);
export const approveEdge = one(Graph.ApproveEdge);
export const rejectEdge = one(Graph.RejectEdge);
export const deleteEdge = one(Graph.DeleteEdge);
export const listEdges = listed<Parameters<typeof Graph.ListEdges>[0], Edge>(Graph.ListEdges);
export const recomputeScores = one(Graph.RecomputeScores);
export const proposeFromPages = one(Graph.ProposeFromPages);
export const proposeRelated = one(Graph.ProposeRelated);
