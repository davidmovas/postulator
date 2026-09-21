import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { useSiteOverview } from "../../data/hooks/reports.js";
import { useSyncSite } from "../../data/hooks/sync.js";
import {
    AccountTreeIcon,
    AddIcon,
    Button,
    CountBadge,
    Screen,
    Segmented,
    TableRowsIcon,
} from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { PlanPageDialog } from "./create.js";
import { PageDrawer } from "./drawer.js";
import { useEntityIndex } from "./entities.js";
import { readQuery, readTab, searchOf, wantsNew, withTab, writeQuery } from "./params.js";
import type { PageTab, PagesQuery, PagesView } from "./params.js";
import { PageRail } from "./rail.js";
import { PageSummary } from "./summary.js";
import { PageTable } from "./table.js";
import { PageToolbar } from "./toolbar.js";
import { PageTree } from "./tree.js";
import { useTreeView } from "./tree-state.js";

const views: readonly SegmentedOption<PagesView>[] = [
    { value: "table", label: copy.pages.views.table, icon: TableRowsIcon },
    { value: "tree", label: copy.pages.views.tree, icon: AccountTreeIcon },
];

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
    const [selectedId, setSelectedId] = useState<string | null>(pageId);
    const [creating, setCreating] = useState(false);
    const asked = wantsNew(searchParams);
    const isTree = query.view === "tree";
    const tree = useTreeView(siteId, isTree, selectedId);

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

    const change = (next: PagesQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const open = (id: string): void => {
        setSelectedId(id);
        void navigate(`/s/${siteId}/pages/${id}${search}`);
    };

    const total = overview.data?.pages.total;

    return (
        <Screen
            title={copy.pages.title}
            badge={total === undefined ? undefined : <CountBadge tone="muted" count={total} />}
            variant="split"
            actions={
                <>
                    <Segmented
                        label={copy.pages.views.table}
                        value={query.view}
                        options={views}
                        onValueChange={(view) => {
                            change({ ...query, view });
                        }}
                    />
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
                </>
            }
            toolbar={
                <PageToolbar
                    query={query}
                    disabled={isTree}
                    tree={
                        isTree && tree.armed
                            ? { nodes: tree.nodes, onExpandAll: tree.expandAll, onCollapseAll: tree.collapseAll }
                            : null
                    }
                    onChange={change}
                />
            }
            left={<PageRail siteId={siteId} query={query} index={index} disabled={isTree} onChange={change} />}
            right={
                selectedId === null ? undefined : (
                    <PageSummary
                        key={selectedId}
                        pageId={selectedId}
                        siteId={siteId}
                        search={search}
                        onOpen={open}
                    />
                )
            }
        >
            {isTree ? (
                <PageTree
                    view={tree}
                    index={index}
                    selectedId={selectedId}
                    onSelect={setSelectedId}
                    onOpen={open}
                    onCreate={() => {
                        setCreating(true);
                    }}
                />
            ) : (
                <PageTable
                    siteId={siteId}
                    query={query}
                    index={index}
                    selectedId={selectedId}
                    onQueryChange={change}
                    onSelect={setSelectedId}
                    onOpen={open}
                    onCreate={() => {
                        setCreating(true);
                    }}
                    onImport={() => {
                        void navigate(`/s/${siteId}/import`);
                    }}
                    onSync={() => {
                        sync.mutate({ siteId });
                    }}
                    syncing={sync.isPending}
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
            <PlanPageDialog
                open={creating}
                onOpenChange={setCreating}
                siteId={siteId}
                index={index}
                search={search}
            />
        </Screen>
    );
}
