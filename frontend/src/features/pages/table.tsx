import type { ReactElement } from "react";
import { useEffect, useMemo } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { usePages } from "../../data/hooks/pages.js";
import { cycleSort } from "../../data/sorts.js";
import type { Page } from "../../data/types.js";
import {
    AccountTreeIcon,
    Button,
    Spinner,
    DenseTable,
    EmptyState,
    FilterAltIcon,
    SkeletonRows,
    SortableHeader,
    TableCell,
    TableHead,
    UploadFileIcon,
    VirtualRows,
} from "../../ui/index.js";
import { PageStatusBadge } from "./badges.js";
import { CategoryCell, DriftCell, EntityCell, PageTableRow, SyncedCell } from "./cells.js";
import type { EntityIndex } from "./entities.js";
import { defaultQuery, filterOf, narrowed } from "./params.js";
import type { PagesQuery } from "./params.js";

const columns = "minmax(96px,2.4fr) minmax(80px,2fr) 60px 84px minmax(80px,1.5fr) minmax(80px,1.5fr) 36px 72px";
const rowHeight = 28;
const pageSize = 200;

interface PageRowProps {
    page: Page;
    selected: boolean;
    entityName: string | null;
    onSelect: (pageId: string) => void;
    onOpen: (pageId: string) => void;
}

function PageRow({ page, selected, entityName, onSelect, onOpen }: PageRowProps): ReactElement {
    return (
        <PageTableRow page={page} selected={selected} onSelect={onSelect} onOpen={onOpen}>
            <TableCell mono={true} title={page.path}>
                {page.path}
            </TableCell>
            <TableCell muted={true}>{page.title === "" ? copy.pages.untitled : page.title}</TableCell>
            <TableCell mono={true} muted={true}>
                {page.wpType}
            </TableCell>
            <TableCell>
                <PageStatusBadge status={page.status} />
            </TableCell>
            <EntityCell entityId={page.entityId} entityName={entityName} />
            <CategoryCell page={page} />
            <DriftCell drift={page.drift} />
            <SyncedCell at={page.lastSyncedAt} />
        </PageTableRow>
    );
}

export interface PageTableProps {
    siteId: string;
    query: PagesQuery;
    index: EntityIndex;
    selectedId: string | null;
    onQueryChange: (next: PagesQuery) => void;
    onSelect: (pageId: string) => void;
    onOpen: (pageId: string) => void;
    onCreate: () => void;
    onImport: () => void;
    onSync: () => void;
    syncing: boolean;
}

export function PageTable({
    siteId,
    query,
    index,
    selectedId,
    onQueryChange,
    onSelect,
    onOpen,
    onCreate,
    onImport,
    onSync,
    syncing,
}: PageTableProps): ReactElement {
    const filter = useMemo(() => filterOf(siteId, query), [siteId, query]);
    const listed = usePages(filter, query.sort, pageSize);
    const rows = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const firstId = rows.length === 0 ? null : rows[0].id;

    useEffect(() => {
        if (selectedId === null && firstId !== null) {
            onSelect(firstId);
        }
    }, [selectedId, firstId, onSelect]);

    if (listed.isPending) {
        return (
            <div className="min-h-0 flex-1 overflow-auto p-3">
                <SkeletonRows rows={12} label={copy.pages.loading} />
            </div>
        );
    }

    if (rows.length === 0) {
        return (
            <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-6">
                {narrowed(query) ? (
                    <EmptyState
                        icon={FilterAltIcon}
                        title={copy.pages.noMatch}
                        actions={
                            <Button
                                onClick={() => {
                                    onQueryChange({ ...defaultQuery, view: query.view, sort: query.sort });
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
                                <Button variant="primary" icon={UploadFileIcon} onClick={onImport}>
                                    {copy.pages.importSheet}
                                </Button>
                                <Button busy={syncing} onClick={onSync}>
                                    {syncing ? copy.pages.syncing : copy.pages.sync}
                                </Button>
                                <Button variant="ghost" onClick={onCreate}>
                                    {copy.pages.newPage}
                                </Button>
                            </>
                        }
                    />
                )}
            </div>
        );
    }

    return (
        <DenseTable columns={columns} label={copy.pages.title} className="min-h-0 flex-1">
            <TableHead>
                <SortableHeader
                    active={query.sort?.field === "path"}
                    direction={query.sort?.desc === true ? "desc" : "asc"}
                    onToggle={() => {
                        onQueryChange({ ...query, sort: cycleSort(query.sort, "path") });
                    }}
                >
                    {copy.pages.columns.path}
                </SortableHeader>
                <div>{copy.pages.columns.title}</div>
                <div>{copy.pages.columns.wpType}</div>
                <div>{copy.pages.columns.status}</div>
                <div>{copy.pages.columns.entity}</div>
                <div>{copy.pages.columns.categories}</div>
                <div>{copy.pages.columns.drift}</div>
                <div>{copy.pages.columns.synced}</div>
            </TableHead>
            <VirtualRows
                count={rows.length}
                rowHeight={rowHeight}
                scrollKey={`${siteId}:pages`}
                row={(position) => {
                    const page = rows[position];
                    if (page === undefined) {
                        return null;
                    }
                    return (
                        <PageRow
                            page={page}
                            selected={page.id === selectedId}
                            entityName={
                                page.entityId === null ? null : (index.labels.get(page.entityId) ?? null)
                            }
                            onSelect={onSelect}
                            onOpen={onOpen}
                        />
                    );
                }}
                onReachEnd={() => {
                    if (listed.hasNextPage && !listed.isFetchingNextPage) {
                        void listed.fetchNextPage();
                    }
                }}
                footer={
                    listed.isFetchingNextPage ? (
                        <div className="flex h-7 items-center justify-center gap-2 text-2xs text-ink-faint">
                            <Spinner size={12} />
                            {copy.app.loadingMore}
                        </div>
                    ) : null
                }
            />
        </DenseTable>
    );
}
