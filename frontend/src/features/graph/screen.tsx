import type { ReactElement } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useGraph } from "../../data/hooks/graph.js";
import { Banner, Button, EmptyState, HubIcon, PublicIcon, SkeletonRows, UploadFileIcon } from "../../ui/index.js";
import { Controls } from "./canvas/controls.js";
import { Legend } from "./canvas/legend.js";
import { GraphMap } from "./canvas/map.js";
import type { MapHandle } from "./canvas/map.js";
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
import { useGraphSession } from "./state.js";
import { Toolbar } from "./toolbar.js";

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
    const map = useRef<MapHandle | null>(null);

    const index = useMemo(() => buildGraphIndex(graph.data?.entities ?? [], graph.data?.edges ?? []), [graph.data]);
    const fold: FoldState = useMemo(() => session.fold ?? defaultFold(index), [session.fold, index]);

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

    const onLens = (lens: Lens): void => {
        change({ ...query, lens });
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
                onView={(view) => {
                    change({ ...query, view });
                }}
                onPick={revealEntity}
                focusSearch={focusSearch}
            />
            <LensBar query={query} counts={counts} onChange={change} />
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
                            onToggle={(id) => {
                                setFold(toggle(fold, id));
                            }}
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
                                revealVersion={revealVersion}
                                onSelect={select}
                                onToggle={(id) => {
                                    setFold(toggle(fold, id));
                                }}
                                onLiftMore={(parentId, count) => {
                                    setFold(liftMore(fold, parentId, count));
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
                                onToggle={() => {
                                    patchSession({ legend: !session.legend });
                                }}
                            />
                        </>
                    )}
                </div>
                {empty || graph.isPending ? null : (
                    <Inspector siteId={siteId} index={index} selectedId={entityId} onSelect={select} onReveal={revealEntity} onLens={onLens} />
                )}
            </div>
        </div>
    );
}
