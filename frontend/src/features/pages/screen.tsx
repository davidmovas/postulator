import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { useSiteOverview } from "../../data/hooks/reports.js";
import { copy } from "../../copy/index.js";
import {
    AccountTreeIcon,
    AddIcon,
    Button,
    EmptyState,
    PublicIcon,
    Segmented,
    TableRowsIcon,
} from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { PlanPageDialog } from "./create.js";
import { PageDrawer } from "./drawer.js";
import { useEntityIndex } from "./entities.js";
import { PageFilters } from "./filters.js";
import { readQuery, searchOf, writeQuery } from "./params.js";
import type { PagesQuery, PagesView } from "./params.js";
import { PageTable } from "./table.js";
import { PageTree } from "./tree.js";

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
    const [planning, setPlanning] = useState(false);

    const change = (next: PagesQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    if (siteId === "") {
        return (
            <div className="flex h-full items-start justify-center p-6">
                <EmptyState
                    icon={PublicIcon}
                    title={copy.shell.noSiteSelected}
                    body={copy.empty.sites}
                />
            </div>
        );
    }

    const total = overview.data?.pages.total;

    return (
        <div className="flex h-full min-h-0">
            {query.view === "table" ? (
                <PageFilters siteId={siteId} query={query} onChange={change} index={index} />
            ) : null}
            <div className="flex min-w-0 flex-1 flex-col">
                <header className="flex h-8 shrink-0 items-center justify-between gap-3 border-b border-hairline px-3">
                    <div className="flex min-w-0 items-center gap-2">
                        <Segmented
                            label={copy.nav.pages}
                            value={query.view}
                            options={views}
                            onValueChange={(view) => {
                                change({ ...query, view });
                            }}
                        />
                        {query.view === "tree" ? (
                            <span className="truncate text-2xs text-ink-faint">{copy.pages.tree.noFilters}</span>
                        ) : null}
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                        {total === undefined ? null : (
                            <span className="font-mono text-2xs text-ink-faint">
                                {copy.pages.siteTotal(total)}
                            </span>
                        )}
                        <Button
                            size="sm"
                            variant="primary"
                            icon={AddIcon}
                            onClick={() => {
                                setPlanning(true);
                            }}
                        >
                            {copy.pages.plan}
                        </Button>
                    </div>
                </header>
                {query.view === "table" ? (
                    <PageTable
                        siteId={siteId}
                        query={query}
                        onQueryChange={change}
                        selectedId={pageId}
                        index={index}
                        search={search}
                        onPlan={() => {
                            setPlanning(true);
                        }}
                    />
                ) : (
                    <PageTree
                        siteId={siteId}
                        query={query}
                        onQueryChange={change}
                        selectedId={pageId}
                        index={index}
                        search={search}
                        onPlan={() => {
                            setPlanning(true);
                        }}
                    />
                )}
            </div>
            {pageId === null ? null : (
                <PageDrawer
                    key={pageId}
                    pageId={pageId}
                    siteId={siteId}
                    index={index}
                    search={search}
                    onClose={() => {
                        void navigate(`/s/${siteId}/pages${search}`);
                    }}
                />
            )}
            <PlanPageDialog
                open={planning}
                onOpenChange={setPlanning}
                siteId={siteId}
                index={index}
                search={search}
            />
        </div>
    );
}
