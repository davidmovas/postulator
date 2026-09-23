import type { ReactElement } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useGraph, useRecomputeScores } from "../../data/hooks/graph.js";
import { useLinkAudit } from "../../data/hooks/reports.js";
import { pushToast } from "../../data/toasts.js";
import {
    AddLinkIcon,
    Banner,
    Button,
    CountBadge,
    EmptyState,
    HubIcon,
    Screen,
    Segmented,
    SkeletonRows,
    TableRowsIcon,
    UploadFileIcon,
} from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { tintByEntity } from "../links/model/audit.js";
import type { EntityIndex } from "../pages/entities.js";
import { GraphActions } from "./actions.js";
import { EntityContextMenu } from "./actions/context-menu.js";
import type { MenuTarget } from "./actions/context-menu.js";
import type { MoveRequest } from "./actions/move-entity.js";
import { Controls } from "./canvas/controls.js";
import { Legend } from "./canvas/legend.js";
import { GraphMap, pulseMs } from "./canvas/map.js";
import type { MapHandle, Pulse } from "./canvas/map.js";
import { Inspector } from "./inspector/panel.js";
import { diffIndex } from "./model/diff.js";
import { collapseToDepth, defaultFold, expandAll, liftMore, reveal, toggle, visibleRows } from "./model/fold.js";
import type { FoldState } from "./model/fold.js";
import { buildGraphIndex } from "./model/index.js";
import { isolate, matchedSet } from "./model/lens.js";
import type { Lens } from "./model/lens.js";
import { readQuery, searchOf, writeQuery } from "./model/params.js";
import type { GraphQuery, GraphView } from "./model/params.js";
import { pruneRows } from "./model/prune.js";
import { NodeCard } from "./node-card.js";
import type { Hovered } from "./node-card.js";
import { OutlineView } from "./outline/view.js";
import { GraphOverlays } from "./overlays.js";
import type { Creating, Proposing } from "./overlays.js";
import { ProposalQueue } from "./queue/sheet.js";
import { useGraphSession } from "./state.js";
import { GraphToolbar } from "./toolbar.js";

const viewOptions: readonly SegmentedOption<GraphView>[] = [
    { value: "map", label: copy.graph.views.map, icon: HubIcon },
    { value: "outline", label: copy.graph.views.outline, icon: TableRowsIcon },
];

interface Card extends Hovered {
    width: number;
}

export function GraphScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const entityId = params.entityId ?? null;
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const search = searchOf(query);
    const graph = useGraph(siteId === "" ? null : siteId);
    const [session, patchSession] = useGraphSession(siteId);
    const [revealVersion, setRevealVersion] = useState(0);
    const [focusSearch, setFocusSearch] = useState(0);
    const [creating, setCreating] = useState<Creating | null>(null);
    const [connectFrom, setConnectFrom] = useState<string | null>(null);
    const [connectTo, setConnectTo] = useState<string | null>(null);
    const [deleting, setDeleting] = useState<string | null>(null);
    const [planning, setPlanning] = useState<string | null>(null);
    const [menu, setMenu] = useState<MenuTarget | null>(null);
    const [hoverEdge, setHoverEdge] = useState<string | null>(null);
    const [proposing, setProposing] = useState<Proposing>(null);
    const [pulse, setPulse] = useState<Pulse | null>(null);
    const [moving, setMoving] = useState<MoveRequest | null>(null);
    const [card, setCard] = useState<Card | null>(null);
    const map = useRef<MapHandle | null>(null);
    const host = useRef<HTMLDivElement>(null);
    const recompute = useRecomputeScores();
    const audit = useLinkAudit(query.proof && siteId !== "" ? siteId : null);
    const auditPages = useMemo(() => (query.proof ? (audit.data?.pages ?? null) : null), [query.proof, audit.data]);
    const tint = useMemo(() => (auditPages === null ? null : tintByEntity(auditPages)), [auditPages]);

    const index = useMemo(() => buildGraphIndex(graph.data?.entities ?? [], graph.data?.edges ?? [], graph.data?.pages ?? []), [graph.data]);
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
        () => ({ entities: index.entities, byId: index.byId, complete: true, loading: graph.isPending }),
        [index, graph.isPending],
    );

    useEffect(() => {
        if (session.fold === null && graph.data !== undefined && index.counts.total > 0) {
            patchSession({ fold: defaultFold(index) });
        }
    }, [session.fold, graph.data, index, patchSession]);

    const asked = searchParams.get("action");

    useEffect(() => {
        if (asked !== "new") {
            return;
        }
        setCreating({ parentId: null });
        setSearchParams(writeQuery(query), { replace: true });
    }, [asked, query, setSearchParams]);

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

    const kinds = useMemo(() => new Set(query.kinds), [query.kinds]);
    const lensActive = query.lens !== "all" || kinds.size > 0;
    const matched = useMemo(() => (lensActive ? matchedSet(index, query.lens, kinds, query.states) : null), [index, query.lens, kinds, lensActive]);
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

    const failure = graph.error === null ? null : react(graph.error);
    const empty = graph.data !== undefined && index.counts.total === 0;
    const connecting = connectFrom === null ? undefined : index.byId.get(connectFrom);
    const ready = !empty && !graph.isPending;

    const content = graph.isPending ? (
        <div className="p-4">
            <SkeletonRows rows={8} label={copy.graph.loading} />
        </div>
    ) : failure !== null && failure.kind !== "silent" && failure.kind !== "unlock" ? (
        <div className="p-4">
            <Banner
                tone="danger"
                title={failure.message}
                actions={
                    <Button size="sm" variant="secondary" onClick={() => void graph.refetch()}>
                        {copy.app.retry}
                    </Button>
                }
            />
        </div>
    ) : empty ? (
        <div className="flex h-full items-center justify-center">
            <EmptyState
                icon={HubIcon}
                title={copy.graph.empty.title}
                body={copy.graph.empty.body}
                actions={
                    <>
                        <Button
                            variant="primary"
                            onClick={() => {
                                setProposing("pages");
                            }}
                        >
                            {copy.graph.empty.propose}
                        </Button>
                        <Button
                            variant="secondary"
                            icon={UploadFileIcon}
                            onClick={() => {
                                void navigate(`/s/${siteId}/import`);
                            }}
                        >
                            {copy.graph.empty.import}
                        </Button>
                    </>
                }
            />
        </div>
    ) : query.view === "outline" ? (
        <OutlineView
            siteId={siteId}
            index={index}
            rows={rows}
            selectedId={entityId}
            matched={matched}
            onSelect={select}
            onPick={pick}
            onToggle={toggleNode}
            onLiftMore={(parentId, count) => {
                setFold(liftMore(fold, parentId, count));
            }}
        />
    ) : (
        <>
            <GraphMap
                siteId={siteId}
                index={index}
                rows={rows}
                fold={fold}
                selectedId={entityId}
                matched={matched}
                showRelated={session.showRelated}
                highlightEdgeId={hoverEdge}
                pulse={pulse}
                tint={tint}
                revealVersion={revealVersion}
                onSelect={select}
                onPick={pick}
                onToggle={toggleNode}
                onLiftMore={(parentId, count) => {
                    setFold(liftMore(fold, parentId, count));
                }}
                onContextMenu={(id, at) => {
                    setCard(null);
                    if (id === null) {
                        setMenu(null);
                        return;
                    }
                    const row = rows.find((held) => held.id === id);
                    setMenu({
                        id,
                        at,
                        expanded: row?.kind === "entity" && row.expanded,
                        hasChildren: row?.kind === "entity" && row.childCount > 0,
                    });
                }}
                onHover={(id, at) => {
                    setCard(id === null ? null : { id, at, width: host.current?.clientWidth ?? 0 });
                }}
                onCreateChild={(parentId) => {
                    setCreating({ parentId });
                }}
                onConnect={startConnect}
                onDelete={setDeleting}
                onReparent={(childId, parentId) => {
                    setMoving({ childId, parentId });
                }}
                ref={(handle) => {
                    map.current = handle;
                }}
                minimap={session.minimap}
            />
            <Controls
                siteId={siteId}
                showRelated={session.showRelated}
                onZoom={(factor) => map.current?.zoomBy(factor)}
                onFit={() => map.current?.fit()}
                onExpandAll={() => {
                    setFold(expandAll(index));
                }}
                onCollapse={() => {
                    setFold(collapseToDepth(index, 1));
                }}
                onToggleRelated={() => {
                    patchSession({ showRelated: !session.showRelated });
                }}
            />
            <Legend
                open={session.legend}
                proof={query.proof}
                onToggle={() => {
                    patchSession({ legend: !session.legend });
                }}
            />
            {card === null || menu !== null ? null : <NodeCard index={index} hovered={card} hostWidth={card.width} />}
            <EntityContextMenu
                target={menu}
                onClose={() => {
                    setMenu(null);
                }}
                onAddChild={(id) => {
                    setCreating({ parentId: id });
                }}
                onConnect={startConnect}
                onToggle={toggleNode}
                onCenter={revealEntity}
                onDelete={setDeleting}
            />
        </>
    );

    return (
        <Screen
            title={copy.graph.title}
            badge={graph.data === undefined ? undefined : <CountBadge tone="muted" count={index.counts.total} />}
            tabs={
                <Segmented
                    label={copy.graph.views.label}
                    value={query.view}
                    options={viewOptions}
                    onValueChange={(view) => {
                        change({ ...query, view });
                    }}
                />
            }
            actions={
                <GraphActions
                    index={index}
                    selectedId={entityId}
                    connecting={connectFrom !== null}
                    reviewing={session.queue}
                    recomputing={recompute.isPending}
                    onCreate={() => {
                        setCreating({ parentId: entityId });
                    }}
                    onConnect={() => {
                        if (entityId !== null) {
                            startConnect(entityId);
                        }
                    }}
                    onStopConnect={stopConnect}
                    onReview={() => {
                        patchSession({ queue: !session.queue });
                    }}
                    onProposeFromPages={() => {
                        setProposing("pages");
                    }}
                    onProposeRelated={() => {
                        setProposing("related");
                    }}
                    onRecompute={recomputeScores}
                />
            }
            toolbar={
                ready ? (
                    <GraphToolbar
                        index={index}
                        query={query}
                        proofBusy={query.proof && audit.isPending}
                        focusSearch={focusSearch}
                        onChange={change}
                        onPick={revealEntity}
                    />
                ) : undefined
            }
            variant="full"
            right={
                ready ? (
                    <Inspector
                        siteId={siteId}
                        index={index}
                        selectedId={entityId}
                        onSelect={select}
                        onReveal={revealEntity}
                        onLens={onLens}
                        onConnect={startConnect}
                        onDelete={setDeleting}
                        onPlanPage={setPlanning}
                        onRecompute={recomputeScores}
                        recomputing={recompute.isPending}
                        audit={auditPages}
                    />
                ) : undefined
            }
        >
            <div
                className="flex h-full min-h-0 flex-col"
                onKeyDown={(event) => {
                    if ((event.ctrlKey || event.metaKey) && (event.key === "f" || event.key === "F")) {
                        event.preventDefault();
                        setFocusSearch((held) => held + 1);
                    }
                }}
            >
                {connecting === undefined ? null : (
                    <div className="border-b border-hairline px-4 py-2">
                        <Banner
                            tone="info"
                            icon={AddLinkIcon}
                            title={copy.graph.connect.picking(connecting.name)}
                            actions={
                                <Button size="sm" variant="ghost" onClick={stopConnect}>
                                    {copy.graph.connect.stop}
                                </Button>
                            }
                        />
                    </div>
                )}
                <div ref={host} className="relative flex min-h-0 flex-1 flex-col">
                    {content}
                    {ready ? (
                        <ProposalQueue
                            index={index}
                            open={session.queue}
                            height={session.queueHeight}
                            onHeightChange={(next) => {
                                patchSession({ queueHeight: next });
                            }}
                            onClose={() => {
                                patchSession({ queue: false });
                                setHoverEdge(null);
                            }}
                            onHover={setHoverEdge}
                            onReveal={revealEntity}
                        />
                    ) : null}
                </div>
            </div>
            <GraphOverlays
                siteId={siteId}
                index={index}
                entityIndex={entityIndex}
                selectedId={entityId}
                creating={creating}
                connectFrom={connectFrom}
                connectTo={connectTo}
                deleting={deleting}
                moving={moving}
                proposing={proposing}
                planning={planning}
                onCreatingChange={setCreating}
                onStopConnect={stopConnect}
                onDeletingChange={setDeleting}
                onMovingChange={setMoving}
                onProposingChange={setProposing}
                onPlanningChange={setPlanning}
                onSelect={select}
                onReviewQueue={openQueue}
            />
        </Screen>
    );
}
