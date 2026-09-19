import type { ReactElement } from "react";
import { useMemo } from "react";
import { useNavigate } from "react-router";

import { flatten } from "../../data/call.js";
import { usePages } from "../../data/hooks/pages.js";
import type { Page } from "../../data/types.js";
import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    AccountTreeIcon,
    Button,
    DenseTable,
    EmptyState,
    FilterAltIcon,
    LinkOffIcon,
    SkeletonRows,
    SortableHeader,
    StatusBadge,
    SyncProblemIcon,
    TableCell,
    TableHead,
    TableRow,
} from "../../ui/index.js";
import type { EntityIndex } from "./entities.js";
import { statusTone } from "./labels.js";
import { defaultQuery, filterOf, narrowed, nextSort } from "./params.js";
import type { PagesQuery } from "./params.js";
import { VirtualRows } from "./virtual-rows.js";

const columns = "minmax(120px,2.4fr) minmax(96px,2fr) 76px 92px minmax(96px,1.5fr) 56px 84px 84px";
const rowHeight = 28;
const pageSize = 200;

interface PageRowProps {
    page: Page;
    selected: boolean;
    entityName: string | null;
    onOpen: (pageId: string) => void;
}

function PageRow({ page, selected, entityName, onOpen }: PageRowProps): ReactElement {
    return (
        <TableRow
            interactive={true}
            selected={selected}
            tabIndex={0}
            onClick={() => {
                onOpen(page.id);
            }}
            onKeyDown={(event) => {
                if (event.key === "Enter") {
                    event.preventDefault();
                    onOpen(page.id);
                }
            }}
        >
            <TableCell mono={true}>{page.path}</TableCell>
            <TableCell muted={true}>{page.title === "" ? copy.pages.untitled : page.title}</TableCell>
            <TableCell mono={true} muted={true}>
                {page.wpType}
            </TableCell>
            <TableCell>
                <StatusBadge tone={statusTone(page.status)}>{page.status}</StatusBadge>
            </TableCell>
            <TableCell muted={page.entityId === null}>
                {page.entityId === null ? (
                    <span className="flex min-w-0 items-center gap-1 text-ink-faint">
                        <LinkOffIcon size={13} className="shrink-0" />
                        <span className="truncate">{copy.pages.unmapped}</span>
                    </span>
                ) : (
                    (entityName ?? copy.pages.mapped)
                )}
            </TableCell>
            <TableCell>
                {page.drift ? (
                    <span className="flex items-center gap-1 text-warn" title={copy.pages.drift.title}>
                        <SyncProblemIcon size={14} className="shrink-0" />
                        <span className="sr-only">{copy.pages.drift.badge}</span>
                    </span>
                ) : null}
            </TableCell>
            <TableCell mono={true} muted={true} title={absoluteTime(page.lastSyncedAt)}>
                {relativeTime(page.lastSyncedAt)}
            </TableCell>
            <TableCell mono={true} muted={true} title={absoluteTime(page.createdAt)}>
                {relativeTime(page.createdAt)}
            </TableCell>
        </TableRow>
    );
}

export interface PageTableProps {
    siteId: string;
    query: PagesQuery;
    onQueryChange: (next: PagesQuery) => void;
    selectedId: string | null;
    index: EntityIndex;
    search: string;
    onPlan: () => void;
}

export function PageTable({
    siteId,
    query,
    onQueryChange,
    selectedId,
    index,
    search,
    onPlan,
}: PageTableProps): ReactElement {
    const navigate = useNavigate();
    const filter = useMemo(() => filterOf(siteId, query), [siteId, query]);
    const listed = usePages(filter, query.sort, pageSize);
    const rows = useMemo(() => flatten(listed.data?.pages), [listed.data]);

    const open = (pageId: string): void => {
        void navigate(`/s/${siteId}/pages/${pageId}${search}`);
    };

    const toggle = (field: "path" | "createdAt"): void => {
        onQueryChange({ ...query, sort: nextSort(query.sort, field) });
    };

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
                        body={copy.pages.noMatchBody}
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
                        actions={
                            <Button variant="primary" onClick={onPlan}>
                                {copy.pages.plan}
                            </Button>
                        }
                    />
                )}
            </div>
        );
    }

    return (
        <DenseTable columns={columns} label={copy.nav.pages} className="min-h-0 flex-1">
            <TableHead>
                <SortableHeader
                    active={query.sort?.field === "path"}
                    direction={query.sort?.desc === true ? "desc" : "asc"}
                    onToggle={() => {
                        toggle("path");
                    }}
                >
                    {copy.pages.columns.path}
                </SortableHeader>
                <div>{copy.pages.columns.title}</div>
                <div>{copy.pages.columns.wpType}</div>
                <div>{copy.pages.columns.status}</div>
                <div>{copy.pages.columns.entity}</div>
                <div>{copy.pages.columns.drift}</div>
                <div>{copy.pages.columns.synced}</div>
                <SortableHeader
                    active={query.sort?.field === "createdAt"}
                    direction={query.sort?.desc === true ? "desc" : "asc"}
                    onToggle={() => {
                        toggle("createdAt");
                    }}
                >
                    {copy.pages.columns.created}
                </SortableHeader>
            </TableHead>
            <VirtualRows
                count={rows.length}
                rowHeight={rowHeight}
                scrollKey={`${siteId}:table`}
                row={(position) => {
                    const page = rows[position];
                    if (page === undefined) {
                        return null;
                    }
                    return (
                        <PageRow
                            page={page}
                            selected={page.id === selectedId}
                            entityName={page.entityId === null ? null : (index.byId.get(page.entityId)?.name ?? null)}
                            onOpen={open}
                        />
                    );
                }}
                footer={
                    listed.hasNextPage ? (
                        <div className="flex justify-center border-t border-hairline p-2">
                            <Button
                                size="sm"
                                busy={listed.isFetchingNextPage}
                                onClick={() => {
                                    void listed.fetchNextPage();
                                }}
                            >
                                {copy.pages.loadMore}
                            </Button>
                        </div>
                    ) : null
                }
            />
        </DenseTable>
    );
}
