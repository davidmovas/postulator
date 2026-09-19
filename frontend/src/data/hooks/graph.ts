import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import type { Cursor, List } from "../../lib/paging.js";
import {
    addEdge,
    approveEdge,
    createEntity,
    deleteEdge,
    deleteEntity,
    getEntity,
    listEdges,
    listEntities,
    loadGraph,
    proposeFromPages,
    proposeRelated,
    recomputeScores,
    rejectEdge,
    setAnchors,
    updateEntity,
} from "../endpoints/graph.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { EntitySort } from "../sorts.js";
import type { Edge, EdgeFilter, Entity, EntityFilter } from "../types.js";

type EdgePages = InfiniteData<List<Edge>, Cursor | undefined>;
type FullGraph = Awaited<ReturnType<typeof loadGraph>>;

export function useGraph(siteId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.graph.full(siteId ?? ""),
        queryFn: ({ signal }) => loadGraph({ siteId: siteId ?? "" }, signal),
        enabled: siteId !== null && siteId !== "",
    });
}

export function useEntities(filter: EntityFilter, sort: EntitySort | null = null, limit?: number) {
    return useUnlockedInfinite<EntityFilter, Entity>({
        queryKey: keys.graph.entities(filter, sort, limit),
        fetch: listEntities,
        filters: filter,
        sort,
        limit,
        enabled: filter.siteId !== "",
    });
}

export function useEdges(filter: EdgeFilter, limit?: number) {
    return useUnlockedInfinite<EdgeFilter, Edge>({
        queryKey: keys.graph.edges(filter, limit),
        fetch: listEdges,
        filters: filter,
        sort: null,
        limit,
        enabled: filter.siteId !== "",
    });
}

export function useEntity(id: string | null) {
    return useUnlockedQuery({
        queryKey: keys.graph.entity(id ?? ""),
        queryFn: ({ signal }) => getEntity({ id: id ?? "" }, signal),
        enabled: id !== null && id !== "",
    });
}

function useEntityWrite<Request, Answer extends { entity: Entity }>(
    call: (request: Request) => Promise<Answer>,
) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        onSuccess: (answered) => {
            client.setQueryData(keys.graph.entity(answered.entity.id), answered);
            void client.invalidateQueries({ queryKey: keys.graph.entityLists() });
            void client.invalidateQueries({ queryKey: keys.graph.full(answered.entity.siteId) });
        },
    });
}

export function useCreateEntity() {
    return useEntityWrite((request: Parameters<typeof createEntity>[0]) => createEntity(request));
}

export function useUpdateEntity() {
    return useEntityWrite((request: Parameters<typeof updateEntity>[0]) => updateEntity(request));
}

export function useSetAnchors() {
    return useEntityWrite((request: Parameters<typeof setAnchors>[0]) => setAnchors(request));
}

export function useDeleteEntity() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteEntity>[0]) => deleteEntity(request),
        onSuccess: (_answered, request) => {
            client.removeQueries({ queryKey: keys.graph.entity(request.id) });
            void client.invalidateQueries({ queryKey: keys.graph.root() });
        },
    });
}

export function useAddEdge() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof addEdge>[0]) => addEdge(request),
        onSuccess: (answered) => {
            void client.invalidateQueries({ queryKey: keys.graph.edgeLists() });
            void client.invalidateQueries({ queryKey: keys.graph.full(answered.edge.siteId) });
        },
    });
}

export function useDeleteEdge() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteEdge>[0]) => deleteEdge(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.graph.edgeLists() });
            void client.invalidateQueries({ queryKey: keys.graph.fulls() });
        },
    });
}

function patchEdgeStatus(client: QueryClient, id: string, status: string): void {
    client.setQueriesData<EdgePages>({ queryKey: keys.graph.edgeLists() }, (held) => {
        if (held === undefined) {
            return held;
        }
        return {
            ...held,
            pages: held.pages.map((loaded) => ({
                ...loaded,
                items: loaded.items.map((edge) => (edge.id === id ? { ...edge, status } : edge)),
            })),
        };
    });
}

function patchFullStatus(client: QueryClient, id: string, status: string): void {
    client.setQueriesData<FullGraph>({ queryKey: keys.graph.fulls() }, (held) => {
        if (held === undefined) {
            return held;
        }
        return { ...held, edges: (held.edges ?? []).map((edge) => (edge.id === id ? { ...edge, status } : edge)) };
    });
}

function useEdgeDecision(call: (request: { id: string }) => Promise<{ edge: Edge }>, status: string) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        onMutate: async (request: { id: string }) => {
            await Promise.all([
                client.cancelQueries({ queryKey: keys.graph.edgeLists() }),
                client.cancelQueries({ queryKey: keys.graph.fulls() }),
            ]);
            const snapshot = [
                ...client.getQueriesData<EdgePages>({ queryKey: keys.graph.edgeLists() }),
                ...client.getQueriesData<FullGraph>({ queryKey: keys.graph.fulls() }),
            ];
            patchEdgeStatus(client, request.id, status);
            patchFullStatus(client, request.id, status);
            return { snapshot };
        },
        onError: (_thrown, _request, context) => {
            for (const [key, held] of context?.snapshot ?? []) {
                client.setQueryData(key, held);
            }
        },
        onSettled: () => {
            void client.invalidateQueries({ queryKey: keys.graph.edgeLists() });
            void client.invalidateQueries({ queryKey: keys.graph.fulls() });
        },
    });
}

export function useApproveEdge() {
    return useEdgeDecision((request) => approveEdge(request), "approved");
}

export function useRejectEdge() {
    return useEdgeDecision((request) => rejectEdge(request), "rejected");
}

export interface BatchInput<Request> {
    request: Request;
    signal?: AbortSignal;
}

function useGraphBatch<Request extends { siteId: string }, Answer>(
    call: (request: Request, signal?: AbortSignal) => Promise<Answer>,
) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (input: BatchInput<Request>) => call(input.request, input.signal),
        onSettled: (_answered, _thrown, input) => {
            void client.invalidateQueries({ queryKey: keys.graph.root() });
            void client.invalidateQueries({ queryKey: keys.pages.lists() });
            void client.invalidateQueries({ queryKey: keys.reports.site(input.request.siteId) });
        },
    });
}

export function useProposeFromPages() {
    return useGraphBatch((request: Parameters<typeof proposeFromPages>[0], signal?: AbortSignal) => proposeFromPages(request, signal));
}

export function useProposeRelated() {
    return useGraphBatch((request: Parameters<typeof proposeRelated>[0], signal?: AbortSignal) => proposeRelated(request, signal));
}

export function useRecomputeScores() {
    return useGraphBatch((request: Parameters<typeof recomputeScores>[0], signal?: AbortSignal) => recomputeScores(request, signal));
}
