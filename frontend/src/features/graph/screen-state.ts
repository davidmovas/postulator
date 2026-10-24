import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useCardProposals, useGraph, useRecomputeScores } from "../../data/hooks/graph.js";
import { useLinkAudit } from "../../data/hooks/reports.js";
import { usePageTypes } from "../../data/hooks/templates.js";
import { pushToast } from "../../data/toasts.js";
import { tintByEntity } from "../links/model/audit.js";
import type { EntityIndex } from "../pages/entities.js";
import type { MoveRequest } from "./actions/move-entity.js";
import { pulseMs } from "./canvas/map.js";
import type { Pulse } from "./canvas/map.js";
import { waitingEntities } from "./cards/model.js";
import { diffIndex } from "./model/diff.js";
import { defaultFold, reveal, toggle, visibleRows } from "./model/fold.js";
import type { FoldState } from "./model/fold.js";
import { buildGraphIndex } from "./model/index.js";
import { isolate, matchedSet } from "./model/lens.js";
import type { Lens } from "./model/lens.js";
import { cardsAction, cardsFocusParam, readQuery, searchOf, writeQuery } from "./model/params.js";
import type { GraphQuery } from "./model/params.js";
import { pruneRows } from "./model/prune.js";
import type { Creating, Proposing } from "./overlays.js";
import { useGraphSession } from "./state.js";

export function useGraphScreen() {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const entityId = params.entityId ?? null;
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const search = searchOf(query);
    const graph = useGraph(siteId === "" ? null : siteId);
    const types = usePageTypes(siteId === "" ? null : siteId);
    const cardProposals = useCardProposals(siteId === "" ? null : siteId);
    const proposedCards = useMemo(() => flatten(cardProposals.data?.pages), [cardProposals.data]);
    const waitingCards = useMemo(() => waitingEntities(proposedCards), [proposedCards]);
    const [session, patchSession] = useGraphSession(siteId);
    const [revealVersion, setRevealVersion] = useState(0);
    const [focusSearch, setFocusSearch] = useState(0);
    const [creating, setCreating] = useState<Creating | null>(null);
    const [connectFrom, setConnectFrom] = useState<string | null>(null);
    const [connectTo, setConnectTo] = useState<string | null>(null);
    const [deleting, setDeleting] = useState<string | null>(null);
    const [planning, setPlanning] = useState<string | null>(null);
    const [hoverEdge, setHoverEdge] = useState<string | null>(null);
    const [proposing, setProposing] = useState<Proposing>(null);
    const [pulse, setPulse] = useState<Pulse | null>(null);
    const [moving, setMoving] = useState<MoveRequest | null>(null);
    const recompute = useRecomputeScores();
    const audit = useLinkAudit(query.proof && siteId !== "" ? siteId : null);
    const auditPages = useMemo(() => (query.proof ? (audit.data?.pages ?? null) : null), [query.proof, audit.data]);
    const tint = useMemo(() => (auditPages === null ? null : tintByEntity(auditPages)), [auditPages]);

    const index = useMemo(
        () => buildGraphIndex(graph.data?.entities ?? [], graph.data?.edges ?? [], graph.data?.pages ?? [], types.data?.types ?? []),
        [graph.data, types.data],
    );
    const previous = useRef(index);

    useEffect(() => {
        const before = previous.current;
        previous.current = index;
        if (before === index || before.counts.total === 0 || before.entities[0]?.siteId !== index.entities[0]?.siteId) {
            return;
        }
        const diff = diffIndex(before, index);
        if (diff.touched.length > 0) {
            setPulse({ ids: new Set(diff.touched), until: Date.now() + pulseMs });
        }
    }, [index]);
    const fold: FoldState = useMemo(() => session.fold ?? defaultFold(index), [session.fold, index]);
    const entityIndex: EntityIndex = useMemo(
        () => ({ entities: index.entities, byId: index.byId, labels: index.labels, complete: true, loading: graph.isPending }),
        [index, graph.isPending],
    );

    useEffect(() => {
        if (session.fold === null && graph.data !== undefined && index.counts.total > 0) {
            patchSession({ fold: defaultFold(index) });
        }
    }, [session.fold, graph.data, index, patchSession]);

    const asked = searchParams.get("action");

    const askedFocus = searchParams.get(cardsFocusParam);

    useEffect(() => {
        if (asked === "new") {
            setCreating({ parentId: null });
        } else if (asked === cardsAction) {
            patchSession({ cards: true, cardFocus: askedFocus });
        } else {
            return;
        }
        setSearchParams(writeQuery(query), { replace: true });
    }, [asked, askedFocus, query, setSearchParams, patchSession]);

    const setFold = useCallback(
        (next: FoldState): void => {
            patchSession({ fold: next });
        },
        [patchSession],
    );

    const change = (next: GraphQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const select = useCallback(
        (id: string | null): void => {
            void navigate(id === null ? `/s/${siteId}/graph${search}` : `/s/${siteId}/graph/${id}${search}`, { replace: true });
        },
        [navigate, siteId, search],
    );

    const pick = useCallback(
        (id: string): void => {
            if (connectFrom !== null && id !== connectFrom) {
                setConnectTo(id);
                return;
            }
            select(id);
        },
        [connectFrom, select],
    );

    const revealEntity = useCallback(
        (id: string): void => {
            setFold(reveal(index, fold, id, session.order));
            setRevealVersion((held) => held + 1);
            select(id);
        },
        [index, fold, session.order, setFold, select],
    );

    const presets = useMemo(() => new Set<string>(query.presets), [query.presets]);
    const lensActive = query.lens !== "all" || presets.size > 0;
    const matched = useMemo(() => (lensActive ? matchedSet(index, query.lens, presets, query.states) : null), [index, query.lens, presets, lensActive]);
    const kept = useMemo(() => (query.isolate && matched !== null ? isolate(index, matched) : null), [index, matched, query.isolate]);
    const visible = useMemo(() => visibleRows(index, fold, session.order), [index, fold, session.order]);
    const rows = useMemo(() => pruneRows(visible, kept), [visible, kept]);

    useEffect(() => {
        if (entityId === null || !index.byId.has(entityId)) {
            return;
        }
        if (!visible.some((row) => row.id === entityId)) {
            setFold(reveal(index, fold, entityId, session.order));
            setRevealVersion((held) => held + 1);
        }
    }, [entityId, index, visible, fold, session.order, setFold]);

    useEffect(() => {
        if (connectFrom !== null && !index.byId.has(connectFrom)) {
            setConnectFrom(null);
            setConnectTo(null);
        }
    }, [connectFrom, index]);

    const onLens = (lens: Lens): void => {
        change({ ...query, lens });
    };

    const startConnect = (id: string): void => {
        setConnectFrom(id);
        setConnectTo(null);
        select(id);
    };

    const stopConnect = (): void => {
        setConnectFrom(null);
        setConnectTo(null);
    };

    const toggleNode = (id: string): void => {
        setFold(toggle(fold, id));
    };

    const recomputeScores = (): void => {
        const held = index;
        recompute.mutate(
            { request: { siteId } },
            {
                onSuccess: (answered) => {
                    const scores = answered.scores ?? {};
                    const changed = Object.entries(scores).filter(([id, score]) => held.byId.get(id)?.score !== score).length;
                    pushToast("info", copy.graph.ai.recomputed(changed));
                },
            },
        );
    };

    const openQueue = (): void => {
        patchSession({ queue: true });
    };

    return {
        navigate,
        siteId,
        entityId,
        query,
        graph,
        proposedCards,
        waitingCards,
        session,
        patchSession,
        revealVersion,
        focusSearch,
        setFocusSearch,
        creating,
        setCreating,
        connectFrom,
        connectTo,
        deleting,
        setDeleting,
        planning,
        setPlanning,
        hoverEdge,
        setHoverEdge,
        proposing,
        setProposing,
        pulse,
        moving,
        setMoving,
        recompute,
        audit,
        auditPages,
        tint,
        index,
        fold,
        entityIndex,
        setFold,
        change,
        select,
        pick,
        revealEntity,
        matched,
        rows,
        onLens,
        startConnect,
        stopConnect,
        toggleNode,
        recomputeScores,
        openQueue,
    };
}
