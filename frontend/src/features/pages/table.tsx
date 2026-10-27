import type { ReactElement, ReactNode } from "react";
import { Fragment, useEffect, useMemo } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { usePageCount, usePages } from "../../data/hooks/pages.js";
import type { PageSort } from "../../data/sorts.js";
import { cycleSort } from "../../data/sorts.js";
import type { Page, PageFilter } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    Checkbox,
    DenseTable,
    shiftHeld,
    SkeletonRows,
    SortableHeader,
    Spinner,
    TableCell,
    TableHead,
    toneClasses,
    useWidthOf,
    VirtualRows,
} from "../../ui/index.js";
import type { Selection } from "../../ui/selection.js";
import { isSelected, noSelection, range, selectAll, toggle } from "../../ui/selection.js";
import { presetIcon, presetTone } from "../settings/types/labels.js";
import { PageStatusBadge } from "./badges.js";
import { DriftCell, EntityCell, PageTableRow, SyncedCell } from "./cells.js";
import type { EntityIndex } from "./entities.js";
import { observedH1Line, titleBesideH1 } from "./labels.js";
import { facetsOf } from "./params.js";
import type { KnownRow, ListColumn, ListLayout } from "./views/model.js";
import { headState, listLayout, pageRow, typeReasonLine } from "./views/model.js";
import type { TypeNaming } from "./views/naming.js";

const rowHeight = 28;
const pageSize = 200;

interface RowContext {
    index: EntityIndex;
    naming: TypeNaming;
    layout: ListLayout;
}

function TitleCell({ page }: { page: Page }): ReactElement {
    const h1 = titleBesideH1(page);
    return (
        <TableCell muted={true} title={observedH1Line(page)}>
            {page.title === "" ? copy.pages.untitled : page.title}
            {h1 === null ? null : <span className="ml-2 text-ink-faint">{h1}</span>}
        </TableCell>
    );
}

function TypeCell({ page, naming }: { page: Page; naming: TypeNaming }): ReactElement {
    const type = naming.typeOf(page.pageTypeId);
    const Icon = presetIcon(type?.preset ?? "");
    return (
        <TableCell muted={type === undefined} title={typeReasonLine(page, naming.ruleOf)}>
            <span data-page-type={page.pageTypeId ?? ""} className="flex min-w-0 items-center gap-1">
                <Icon size={13} className={`shrink-0 ${toneClasses[presetTone(type?.preset ?? "")].ink}`} />
                <span className="truncate">{naming.typeName(page.pageTypeId)}</span>
            </span>
        </TableCell>
    );
}

interface PageRowProps {
    page: Page;
    context: RowContext;
    focused: boolean;
    picked: boolean;
    onFocus: (pageId: string) => void;
    onOpen: (pageId: string) => void;
    onPick: (page: Page, ranged: boolean) => void;
}

function PageRow({ page, context, focused, picked, onFocus, onOpen, onPick }: PageRowProps): ReactElement {
    const cellOf = (column: ListColumn): ReactNode => {
        switch (column) {
            case "box":
                return (
                    <span
                        className="flex items-center"
                        onClick={(event) => {
                            event.stopPropagation();
                        }}
                        onMouseDown={(event) => {
                            if (event.shiftKey) {
                                event.preventDefault();
                            }
                        }}
                    >
                        <Checkbox
                            aria-label={copy.pages.select.row(page.path)}
                            checked={picked}
                            onChange={(event) => {
                                onPick(page, shiftHeld(event));
                            }}
                        />
                    </span>
                );
            case "path":
                return (
                    <TableCell mono={true} title={page.path}>
                        {page.path}
                    </TableCell>
                );
            case "title":
                return <TitleCell page={page} />;
            case "type":
                return <TypeCell page={page} naming={context.naming} />;
            case "status":
                return (
                    <TableCell>
                        <PageStatusBadge status={page.status} />
                    </TableCell>
                );
            case "entity":
                return (
                    <EntityCell
                        entityId={page.entityId}
                        entityName={page.entityId === null ? null : (context.index.labels.get(page.entityId) ?? null)}
                    />
                );
            case "created":
                return (
                    <TableCell mono={true} muted={true} title={absoluteTime(page.createdAt)}>
                        {relativeTime(page.createdAt)}
                    </TableCell>
                );
            case "drift":
                return <DriftCell drift={page.drift} />;
            default:
                return <SyncedCell at={page.lastSyncedAt} />;
        }
    };
    return (
        <PageTableRow page={page} selected={focused} onSelect={onFocus} onOpen={onOpen}>
            {context.layout.columns.map((column) => (
                <Fragment key={column}>{cellOf(column)}</Fragment>
            ))}
        </PageTableRow>
    );
}

export interface PageTableProps {
    filter: PageFilter;
    sort: PageSort | null;
    onSort: (sort: PageSort | null) => void;
    index: EntityIndex;
    naming: TypeNaming;
    selection: Selection;
    onSelection: (next: Selection) => void;
    focusedId?: string | null;
    onFocus?: (pageId: string) => void;
    onOpen: (pageId: string) => void;
    onRows?: (rows: ReadonlyMap<string, KnownRow>) => void;
    empty: ReactNode;
    scrollKey: string;
    label: string;
}

export function PageTable({
    filter,
    sort,
    onSort,
    index,
    naming,
    selection,
    onSelection,
    focusedId = null,
    onFocus,
    onOpen,
    onRows,
    empty,
    scrollKey,
    label,
}: PageTableProps): ReactElement {
    const listed = usePages(filter, sort, pageSize);
    const counted = usePageCount(filter);
    const rows = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const ids = useMemo(() => rows.map((page) => page.id), [rows]);
    const byId = useMemo(() => new Map(rows.map((page) => [page.id, page] as const)), [rows]);
    const facets = useMemo(() => facetsOf(filter), [filter]);
    const [attach, width] = useWidthOf();
    const layout = useMemo(() => listLayout(width), [width]);
    const context: RowContext = { index, naming, layout };
    const firstId = rows.length === 0 ? null : (rows[0]?.id ?? null);
    const matching = counted.data?.pages ?? null;
    const head = headState(selection, facets);

    useEffect(() => {
        if (onFocus !== undefined && focusedId === null && firstId !== null) {
            onFocus(firstId);
        }
    }, [focusedId, firstId, onFocus]);

    useEffect(() => {
        onRows?.(new Map(rows.map((page) => [page.id, { path: page.path, row: pageRow(page) }] as const)));
    }, [rows, onRows]);

    if (listed.isPending) {
        return (
            <div className="min-h-0 flex-1 overflow-auto p-3">
                <SkeletonRows rows={12} label={copy.pages.loading} />
            </div>
        );
    }

    if (rows.length === 0) {
        return <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-6">{empty}</div>;
    }

    const pick = (page: Page, ranged: boolean): void => {
        const row = pageRow(page);
        if (ranged) {
            onSelection(
                range(selection, ids, page.id, (id) => {
                    const held = byId.get(id);
                    return held === undefined ? undefined : pageRow(held);
                }),
            );
            return;
        }
        onSelection(toggle(selection, page.id, row) ?? selection);
    };

    const sortable = (field: PageSort["field"], text: string): ReactElement => (
        <SortableHeader
            active={sort?.field === field}
            direction={sort?.desc === true ? "desc" : "asc"}
            onToggle={() => {
                onSort(cycleSort(sort, field));
            }}
        >
            {text}
        </SortableHeader>
    );

    const headOf = (column: ListColumn): ReactNode => {
        switch (column) {
            case "box":
                return (
                    <span className="flex items-center">
                        <Checkbox
                            aria-label={matching === null ? copy.pages.select.allMatching(rows.length) : copy.pages.select.allMatching(matching)}
                            title={matching === null ? undefined : copy.pages.select.allMatching(matching)}
                            checked={head === "on"}
                            indeterminate={head === "some"}
                            disabled={head === "off" && matching === null}
                            onChange={() => {
                                if (head === "off" && matching !== null) {
                                    onSelection(selectAll(selection, facets, matching));
                                } else {
                                    onSelection(noSelection);
                                }
                            }}
                        />
                    </span>
                );
            case "path":
                return sortable("path", copy.pages.columns.path);
            case "title":
                return sortable("title", copy.pages.columns.title);
            case "created":
                return sortable("createdAt", copy.pages.columns.created);
            case "type":
                return <div>{copy.pages.columns.type}</div>;
            case "status":
                return <div>{copy.pages.columns.status}</div>;
            case "entity":
                return <div>{copy.pages.columns.entity}</div>;
            case "drift":
                return <div>{copy.pages.columns.drift}</div>;
            default:
                return <div>{copy.pages.columns.synced}</div>;
        }
    };

    return (
        <div ref={attach} data-page-list={layout.columns.join(" ")} className="flex min-h-0 min-w-0 flex-1 flex-col">
            <DenseTable columns={layout.template} label={label} className="min-h-0 flex-1">
                <TableHead>
                    {layout.columns.map((column) => (
                        <Fragment key={column}>{headOf(column)}</Fragment>
                    ))}
                </TableHead>
                <VirtualRows
                    count={rows.length}
                    rowHeight={rowHeight}
                    scrollKey={scrollKey}
                    row={(position) => {
                        const page = rows[position];
                        if (page === undefined) {
                            return null;
                        }
                        return (
                            <PageRow
                                page={page}
                                context={context}
                                focused={page.id === focusedId}
                                picked={isSelected(selection, page.id, pageRow(page))}
                                onFocus={onFocus ?? onOpen}
                                onOpen={onOpen}
                                onPick={pick}
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
        </div>
    );
}
