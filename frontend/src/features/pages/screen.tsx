import type { ReactElement } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { usePageCount } from "../../data/hooks/pages.js";
import { useSiteOverview } from "../../data/hooks/reports.js";
import { useSyncSite } from "../../data/hooks/sync.js";
import { wantsNew } from "../../data/params.js";
import { AccountTreeIcon, AddIcon, Button, CountBadge, EmptyState, FilterAltIcon, Screen, UploadFileIcon } from "../../ui/index.js";
import type { Selection } from "../../ui/selection.js";
import { dropWhole, isEmpty, selectAll } from "../../ui/selection.js";
import { StartRunDrawer } from "../runs/start.js";
import { PlanPageDialog } from "./create.js";
import { PageDrawer } from "./drawer.js";
import { useEntityIndex } from "./entities.js";
import { defaultQuery, facetsOf, filterOf, narrowed, readQuery, readTab, searchOf, withTab, writeQuery } from "./params.js";
import type { PageTab, PagesQuery } from "./params.js";
import { PageRail } from "./rail.js";
import { PageSummary } from "./summary.js";
import { PageTable } from "./table.js";
import { PageToolbar } from "./toolbar.js";
import { BulkBar } from "./views/bulk.js";
import type { PagesView } from "./views/memory.js";
import { pagesMemoryOf, rememberPages, rememberSelection, usePagesMemory } from "./views/memory.js";
import type { KnownRow } from "./views/model.js";
import { wholeOf } from "./views/model.js";
import { useTypeNaming } from "./views/naming.js";
import { PagesTree } from "./views/tree.js";
import { PagesByType } from "./views/types.js";

export function PagesScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const pageId = params.pageId ?? null;
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const search = searchOf(query);
    const index = useEntityIndex(siteId);
    const overview = useSiteOverview(siteId);
    const sync = useSyncSite();
    const naming = useTypeNaming(siteId);
    const memory = usePagesMemory(siteId);
    const view = memory.view;
    const selection = memory.selections[view];
    const [selectedId, setSelectedId] = useState<string | null>(pageId);
    const [creating, setCreating] = useState(false);
    const [running, setRunning] = useState<readonly string[] | null>(null);
    const [treeExpanded, setTreeExpanded] = useState<ReadonlySet<string>>(new Set());
    const known = useRef(new Map<string, KnownRow>());
    const filter = useMemo(() => filterOf(siteId, query), [siteId, query]);
    const facets = useMemo(() => facetsOf(filter), [filter]);
    const facetsKey = JSON.stringify(facets);
    const counted = usePageCount(view === "list" ? filter : null);
    const asked = wantsNew(searchParams);

    useEffect(() => {
        if (asked) {
            setCreating(true);
            setSearchParams(writeQuery(readQuery(searchParams)), { replace: true });
        }
    }, [asked, searchParams, setSearchParams]);

    useEffect(() => {
        if (pageId !== null) {
            setSelectedId(pageId);
        }
    }, [pageId]);

    useEffect(() => {
        known.current = new Map();
        setTreeExpanded(new Set());
    }, [siteId]);

    useEffect(() => {
        const held = pagesMemoryOf(siteId).selections.list;
        if (held.whole !== null && !wholeOf(held, facets)) {
            rememberSelection(siteId, "list", dropWhole(held));
        }
    }, [siteId, facetsKey]);

    const setSelection = useCallback(
        (next: Selection) => {
            rememberSelection(siteId, view, next);
        },
        [siteId, view],
    );

    const addRows = useCallback((rows: ReadonlyMap<string, KnownRow>) => {
        for (const [id, row] of rows) {
            known.current.set(id, row);
        }
    }, []);

    const change = (next: PagesQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const open = (id: string): void => {
        setSelectedId(id);
        void navigate(`/s/${siteId}/pages/${id}${search}`);
    };

    const under = (id: string): void => {
        rememberPages(siteId, { view: "list" });
        change({ ...query, branch: id });
    };

    const showView = (next: PagesView): void => {
        rememberPages(siteId, { view: next });
    };

    const total = overview.data?.pages.total;
    const matching = counted.data?.pages ?? null;

    const listEmpty = narrowed(query) ? (
        <EmptyState
            icon={FilterAltIcon}
            title={copy.pages.noMatch}
            actions={
                <Button
                    onClick={() => {
                        change({ ...defaultQuery, sort: query.sort });
                    }}
                >
                    {copy.pages.filters.reset}
                </Button>
            }
        />
    ) : (
        <EmptyState
            icon={AccountTreeIcon}
            title={copy.pages.empty.title}
            body={copy.empty.pages}
            className="w-96"
            actions={
                <>
                    <Button
                        variant="primary"
                        icon={UploadFileIcon}
                        onClick={() => {
                            void navigate(`/s/${siteId}/import`);
                        }}
                    >
                        {copy.pages.importSheet}
                    </Button>
                    <Button
                        busy={sync.isPending}
                        onClick={() => {
                            sync.mutate({ siteId });
                        }}
                    >
                        {sync.isPending ? copy.pages.syncing : copy.pages.sync}
                    </Button>
                    <Button
                        variant="ghost"
                        onClick={() => {
                            setCreating(true);
                        }}
                    >
                        {copy.pages.newPage}
                    </Button>
                </>
            }
        />
    );

    const body = ((): ReactElement => {
        switch (view) {
            case "tree":
                return (
                    <PagesTree
                        siteId={siteId}
                        selection={selection}
                        onSelection={setSelection}
                        naming={naming}
                        scrollKey={`${siteId}:pages-tree`}
                        expanded={treeExpanded}
                        onExpanded={setTreeExpanded}
                        empty={<EmptyState icon={AccountTreeIcon} title={copy.pages.tree.empty} />}
                        onRows={addRows}
                        onOpen={open}
                        onUnder={under}
                    />
                );
            case "types":
                return (
                    <PagesByType
                        siteId={siteId}
                        index={index}
                        naming={naming}
                        selection={selection}
                        onSelection={setSelection}
                        openType={memory.openType}
                        mode={memory.groupMode}
                        onOpenType={(typeId) => {
                            rememberPages(siteId, { openType: typeId });
                        }}
                        onMode={(groupMode) => {
                            rememberPages(siteId, { groupMode });
                        }}
                        onRows={addRows}
                        onOpen={open}
                        onUnder={under}
                    />
                );
            default:
                return (
                    <PageTable
                        filter={filter}
                        sort={query.sort}
                        onSort={(sort) => {
                            change({ ...query, sort });
                        }}
                        index={index}
                        naming={naming}
                        selection={selection}
                        onSelection={setSelection}
                        focusedId={selectedId}
                        onFocus={setSelectedId}
                        onOpen={open}
                        onRows={addRows}
                        empty={listEmpty}
                        scrollKey={`${siteId}:pages`}
                        label={copy.pages.title}
                    />
                );
        }
    })();

    return (
        <Screen
            title={copy.pages.title}
            badge={total === undefined ? undefined : <CountBadge tone="muted" count={total} />}
            variant="split"
            actions={
                <Button
                    data-page-create={true}
                    variant="primary"
                    icon={AddIcon}
                    onClick={() => {
                        setCreating(true);
                    }}
                >
                    {copy.pages.newPage}
                </Button>
            }
            toolbar={<PageToolbar view={view} onView={showView} query={query} onChange={change} />}
            left={view === "list" ? <PageRail siteId={siteId} query={query} index={index} naming={naming} onChange={change} /> : undefined}
            right={
                view !== "list" || selectedId === null ? undefined : (
                    <PageSummary key={selectedId} pageId={selectedId} siteId={siteId} search={search} onOpen={open} />
                )
            }
        >
            {body}
            {isEmpty(selection) ? null : (
                <BulkBar
                    siteId={siteId}
                    selection={selection}
                    onSelection={setSelection}
                    naming={naming}
                    known={() => known.current}
                    entityName={(entityId) => index.labels.get(entityId) ?? entityId}
                    selectAll={
                        view === "list" && matching !== null && matching > 0 && !wholeOf(selection, facets)
                            ? {
                                  count: matching,
                                  onSelect: () => {
                                      setSelection(selectAll(selection, facets, matching));
                                  },
                              }
                            : null
                    }
                    onStartRun={setRunning}
                />
            )}

            {pageId === null ? null : (
                <PageDrawer
                    key={pageId}
                    pageId={pageId}
                    siteId={siteId}
                    index={index}
                    search={search}
                    tab={readTab(searchParams)}
                    onTabChange={(tab: PageTab) => {
                        setSearchParams(withTab(searchParams, tab), { replace: true });
                    }}
                    onClose={() => {
                        void navigate(`/s/${siteId}/pages${search}`);
                    }}
                />
            )}
            <PlanPageDialog open={creating} onOpenChange={setCreating} siteId={siteId} index={index} search={search} />
            <StartRunDrawer
                open={running !== null}
                onOpenChange={(next) => {
                    if (!next) {
                        setRunning(null);
                    }
                }}
                siteId={siteId}
                preselect={running ?? []}
                paths={running === null ? undefined : new Map([...known.current].map(([id, held]) => [id, held.path] as const))}
                onStarted={(runId) => {
                    setRunning(null);
                    void navigate(`/s/${siteId}/runs/${runId}`);
                }}
            />
        </Screen>
    );
}
