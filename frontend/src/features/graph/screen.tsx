import type { ReactElement } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useGraph, useRecomputeScores } from "../../data/hooks/graph.js";
import { useLinkAudit } from "../../data/hooks/reports.js";
import { pushToast } from "../../data/toasts.js";
import { AddLinkIcon, Banner, Button, EmptyState, HubIcon, PublicIcon, SkeletonRows, UploadFileIcon } from "../../ui/index.js";
import { tintByEntity } from "../links/model/audit.js";
import { PlanPageDialog } from "../pages/create.js";
import type { EntityIndex } from "../pages/entities.js";
import { ConnectDrawer } from "./actions/connect.js";
import { EntityContextMenu } from "./actions/context-menu.js";
import type { MenuTarget } from "./actions/context-menu.js";
import { CreateEntityDrawer } from "./actions/create-entity.js";
import { DeleteEntityDialog } from "./actions/delete-entity.js";
import { ProposeFromPagesDialog, ProposeRelatedDialog } from "./actions/propose.js";
import { Controls } from "./canvas/controls.js";
import { Legend } from "./canvas/legend.js";
import { GraphMap, pulseMs } from "./canvas/map.js";
import type { MapHandle, Pulse } from "./canvas/map.js";
import { diffIndex } from "./model/diff.js";
import { Inspector } from "./inspector/panel.js";
import { LensBar } from "./lens-bar.js";
import { collapseToDepth, defaultFold, expandAll, liftMore, reveal, toggle, visibleRows } from "./model/fold.js";
import type { FoldState } from "./model/fold.js";
import { buildGraphIndex } from "./model/index.js";
import { isolate, lensCounts, matchedSet } from "./model/lens.js";
import type { Lens } from "./model/lens.js";
import { readQuery, searchOf, writeQuery } from "./model/params.js";
import type { GraphQuery } from "./model/params.js";
import { pruneRows } from "./model/prune.js";
import { OutlineView } from "./outline/view.js";
import { ProposalQueue } from "./queue/sheet.js";
import { useGraphSession } from "./state.js";
import { Toolbar } from "./toolbar.js";

interface Creating {
    parentId: string | null;
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
    const [proposing, setProposing] = useState<"pages" | "related" | null>(null);
    const [pulse, setPulse] = useState<Pulse | null>(null);
    const map = useRef<MapHandle | null>(null);
    const recompute = useRecomputeScores();
    const audit = useLinkAudit(query.proof && siteId !== "" ? siteId : null);
    const auditPages = useMemo(() => (query.proof ? (audit.data?.pages ?? null) : null), [query.proof, audit.data]);
    const tint = useMemo(() => (auditPages === null ? null : tintByEntity(auditPages)), [auditPages]);

    const index = useMemo(() => buildGraphIndex(graph.data?.entities ?? [], graph.data?.edges ?? []), [graph.data]);
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
    const matched = useMemo(() => (lensActive ? matchedSet(index, query.lens, kinds) : null), [index, query.lens, kinds, lensActive]);
    const kept = useMemo(() => (query.isolate && matched !== null ? isolate(index, matched) : null), [index, matched, query.isolate]);
    const visible = useMemo(() => visibleRows(index, fold, session.order), [index, fold, session.order]);
    const rows = useMemo(() => pruneRows(visible, kept), [visible, kept]);
    const counts = useMemo(() => lensCounts(index), [index]);

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

    if (siteId === "") {
        return (
            <div className="flex h-full items-start justify-center p-6">
                <EmptyState icon={PublicIcon} title={copy.shell.noSiteSelected} body={copy.empty.sites} />
            </div>
        );
    }

    const failure = graph.error === null ? null : react(graph.error);
    const empty = graph.data !== undefined && index.counts.total === 0;
    const connecting = connectFrom === null ? undefined : index.byId.get(connectFrom);

    return (
        <div
            className="flex h-full min-h-0 flex-col"
            onKeyDown={(event) => {
                if ((event.ctrlKey || event.metaKey) && (event.key === "f" || event.key === "F")) {
                    event.preventDefault();
                    setFocusSearch((held) => held + 1);
                }
            }}
        >
            <Toolbar
                index={index}
                view={query.view}
                selectedId={entityId}
                connectFrom={connectFrom}
                onView={(view) => {
                    change({ ...query, view });
                }}
                onPick={revealEntity}
                onCreate={() => {
                    setCreating({ parentId: entityId });
                }}
                onConnect={() => {
                    if (entityId !== null) {
                        startConnect(entityId);
                    }
                }}
                onStopConnect={stopConnect}
                reviewing={session.queue}
                recomputing={recompute.isPending}
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
                focusSearch={focusSearch}
            />
            <LensBar query={query} counts={counts} proofBusy={query.proof && audit.isPending} onChange={change} />
            {connecting === undefined ? null : (
                <div className="border-b border-hairline px-3 py-2">
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
            <div className="flex min-h-0 flex-1">
                <div className="relative flex min-w-0 flex-1 flex-col">
                    {graph.isPending ? (
                        <div className="p-3">
                            <SkeletonRows rows={8} label={copy.graph.loading} />
                        </div>
                    ) : failure !== null && failure.kind !== "silent" && failure.kind !== "unlock" ? (
                        <div className="p-3">
                            <Banner tone="danger" title={failure.message} />
                        </div>
                    ) : empty ? (
                        <div className="flex h-full items-start justify-center p-6">
                            <EmptyState
                                icon={HubIcon}
                                title={copy.graph.empty.title}
                                body={copy.graph.empty.body}
                                actions={
                                    <>
                                        <Button
                                            variant="primary"
                                            size="sm"
                                            onClick={() => {
                                                setProposing("pages");
                                            }}
                                        >
                                            {copy.graph.empty.propose}
                                        </Button>
                                        <Button
                                            variant="secondary"
                                            size="sm"
                                            onClick={() => {
                                                setCreating({ parentId: null });
                                            }}
                                        >
                                            {copy.graph.empty.add}
                                        </Button>
                                        <Button
                                            variant="secondary"
                                            size="sm"
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
                                onCreateChild={(parentId) => {
                                    setCreating({ parentId });
                                }}
                                onConnect={startConnect}
                                onDelete={setDeleting}
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
                                proof={tint !== null}
                                onToggle={() => {
                                    patchSession({ legend: !session.legend });
                                }}
                            />
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
                    )}
                    {empty || graph.isPending ? null : (
                        <ProposalQueue
                            index={index}
                            open={session.queue}
                            onClose={() => {
                                patchSession({ queue: false });
                                setHoverEdge(null);
                            }}
                            onHover={setHoverEdge}
                            onReveal={revealEntity}
                        />
                    )}
                </div>
                {empty || graph.isPending ? null : (
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
                        audit={auditPages}
                    />
                )}
            </div>
            <CreateEntityDrawer
                open={creating !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        setCreating(null);
                    }
                }}
                siteId={siteId}
                index={index}
                parentId={creating?.parentId ?? null}
                onCreated={select}
            />
            <ConnectDrawer
                siteId={siteId}
                index={index}
                fromId={connectFrom}
                toId={connectTo}
                onClose={stopConnect}
            />
            <DeleteEntityDialog
                index={index}
                entityId={deleting}
                onClose={() => {
                    setDeleting(null);
                }}
                onDeleted={() => {
                    if (deleting === entityId) {
                        select(null);
                    }
                }}
            />
            <ProposeFromPagesDialog
                open={proposing === "pages"}
                onOpenChange={(open) => {
                    if (!open) {
                        setProposing(null);
                    }
                }}
                siteId={siteId}
                onReview={openQueue}
            />
            <ProposeRelatedDialog
                open={proposing === "related"}
                onOpenChange={(open) => {
                    if (!open) {
                        setProposing(null);
                    }
                }}
                siteId={siteId}
                index={index}
                selectedId={entityId}
                onReview={openQueue}
            />
            <PlanPageDialog
                open={planning !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        setPlanning(null);
                    }
                }}
                siteId={siteId}
                index={entityIndex}
                search=""
                initialEntityId={planning ?? undefined}
            />
        </div>
    );
}
