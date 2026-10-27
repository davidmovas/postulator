import type { InfiniteData, QueryClient, QueryKey } from "@tanstack/react-query";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import type { Cursor, List } from "../../lib/paging.js";
import { quietMeta } from "../client.js";
import {
    acceptCard,
    addEdge,
    adoptParents,
    applyProposals,
    approveEdge,
    createEntity,
    deleteEntity,
    dismissCard,
    getEntity,
    listCardProposals,
    listEdges,
    listEntities,
    loadGraph,
    moveEntity,
    previewFromPages,
    proposeFromKeywords,
    proposeRelated,
    recomputeScores,
    rejectEdge,
    setAnchors,
    updateEntity,
} from "../endpoints/graph.js";
import { keys } from "../keys.js";
import { useLockGate } from "../lock.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { EntitySort } from "../sorts.js";
import type { AdoptParentsResponse, CardProposal, Edge, EdgeFilter, Entity, EntityFilter } from "../types.js";

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

export function useMoveEntity() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof moveEntity>[0]) => moveEntity(request),
        onSuccess: (answered) => {
            void client.invalidateQueries({ queryKey: keys.graph.edgeLists() });
            void client.invalidateQueries({ queryKey: keys.graph.full(answered.edge.siteId) });
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
    alsoMoved: readonly QueryKey[] = [],
) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (input: BatchInput<Request>) => call(input.request, input.signal),
        onSettled: (_answered, _thrown, input) => {
            void client.invalidateQueries({ queryKey: keys.graph.root() });
            void client.invalidateQueries({ queryKey: keys.pages.lists() });
            void client.invalidateQueries({ queryKey: keys.reports.site(input.request.siteId) });
            for (const key of alsoMoved) {
                void client.invalidateQueries({ queryKey: key });
            }
        },
    });
}

const mappingMoves: readonly QueryKey[] = [keys.findings.root(), keys.pages.maps(), keys.pages.models()];

export function useProposeRelated() {
    return useGraphBatch((request: Parameters<typeof proposeRelated>[0], signal?: AbortSignal) => proposeRelated(request, signal));
}

export function usePreviewFromPages() {
    return useMutation({
        mutationFn: (input: BatchInput<Parameters<typeof previewFromPages>[0]>) => previewFromPages(input.request, input.signal),
    });
}

export function useProposeFromKeywords() {
    return useMutation({
        mutationFn: (input: BatchInput<Parameters<typeof proposeFromKeywords>[0]>) => proposeFromKeywords(input.request, input.signal),
    });
}

export function useApplyProposals() {
    return useGraphBatch(
        (request: Parameters<typeof applyProposals>[0], signal?: AbortSignal) => applyProposals(request, signal),
        mappingMoves,
    );
}

export function useRecomputeScores() {
    return useGraphBatch((request: Parameters<typeof recomputeScores>[0], signal?: AbortSignal) => recomputeScores(request, signal));
}

export const adoptionLimit = 100;

export function useAdoptionPreview(siteId: string, under: string, enabled: boolean) {
    const gate = useLockGate();
    return useInfiniteQuery({
        queryKey: keys.graph.adoption(siteId, under),
        enabled: enabled && siteId !== "" && gate.ready && !gate.locked,
        initialPageParam: undefined as Cursor | undefined,
        queryFn: ({ pageParam, signal }) =>
            adoptParents(
                {
                    siteId,
                    limit: adoptionLimit,
                    ...(under === "" ? {} : { under }),
                    ...(pageParam === undefined ? {} : { cursor: pageParam }),
                },
                signal,
            ),
        getNextPageParam: (last: AdoptParentsResponse) =>
            last.steps.hasMore && last.steps.nextCursor !== undefined ? last.steps.nextCursor : undefined,
        getPreviousPageParam: () => undefined,
        refetchOnWindowFocus: false,
        gcTime: 0,
    });
}

export function useAdoptParents() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: { siteId: string; under: string; hash: string }) =>
            adoptParents({
                siteId: request.siteId,
                hash: request.hash,
                apply: true,
                limit: 1,
                ...(request.under === "" ? {} : { under: request.under }),
            }),
        meta: quietMeta(["CONFLICT"]),
        onSettled: (_answered, _thrown, request) => {
            void client.invalidateQueries({ queryKey: keys.graph.root() });
            void client.invalidateQueries({ queryKey: keys.pages.lists() });
            void client.invalidateQueries({ queryKey: keys.reports.site(request.siteId) });
            for (const key of mappingMoves) {
                void client.invalidateQueries({ queryKey: key });
            }
        },
    });
}

export const cardProposalsLimit = 100;

export function useCardProposals(siteId: string | null) {
    return useUnlockedInfinite<{ siteId: string }, CardProposal>({
        queryKey: keys.graph.cards(siteId ?? ""),
        fetch: listCardProposals,
        filters: { siteId: siteId ?? "" },
        sort: null,
        limit: cardProposalsLimit,
        enabled: siteId !== null && siteId !== "",
    });
}

function useCardDecision<Request, Answer>(call: (request: Request) => Promise<Answer>, siteId: string) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        meta: quietMeta(["CONFLICT"]),
        onSettled: () => {
            void client.invalidateQueries({ queryKey: keys.graph.cards(siteId) });
        },
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.graph.entityAll() });
            void client.invalidateQueries({ queryKey: keys.graph.entityLists() });
            void client.invalidateQueries({ queryKey: keys.graph.full(siteId) });
        },
    });
}

export function useAcceptCard(siteId: string) {
    return useCardDecision((request: Parameters<typeof acceptCard>[0]) => acceptCard(request), siteId);
}

export function useDismissCard(siteId: string) {
    return useCardDecision((request: Parameters<typeof dismissCard>[0]) => dismissCard(request), siteId);
}
