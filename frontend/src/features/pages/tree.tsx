import type { ReactElement } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";

import { usePageTree } from "../../data/hooks/pages.js";
import { useSiteOverview } from "../../data/hooks/reports.js";
import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    AccountTreeIcon,
    Banner,
    Button,
    ChevronRightIcon,
    cx,
    DenseTable,
    DescriptionIcon,
    EmptyState,
    LinkOffIcon,
    SkeletonRows,
    StatusBadge,
    SyncProblemIcon,
    TableCell,
    TableHead,
    TableRow,
} from "../../ui/index.js";
import type { EntityIndex } from "./entities.js";
import { statusTone } from "./labels.js";
import type { PagesQuery } from "./params.js";
import { branchIds, countNodes, flattenTree, pathTo } from "./tree-model.js";
import type { TreeRow } from "./tree-model.js";
import { VirtualRows } from "../../ui/index.js";

const columns = "minmax(200px,3fr) 92px minmax(96px,1.5fr) 56px 84px";
const rowHeight = 28;
const indentStep = 14;
const treeFetchLimit = 1500;
const autoExpandLimit = 200;
const childLimit = 200;

interface BranchRowProps {
    row: Extract<TreeRow, { kind: "page" }>;
    selected: boolean;
    entityName: string | null;
    onToggle: (pageId: string) => void;
    onOpen: (pageId: string) => void;
}

function BranchRow({ row, selected, entityName, onToggle, onOpen }: BranchRowProps): ReactElement {
    const { page } = row;
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
            <TableCell>
                <span className="flex min-w-0 items-center gap-1" style={{ paddingLeft: `${row.depth * indentStep}px` }}>
                    {row.childCount === 0 ? (
                        <DescriptionIcon size={13} className="shrink-0 text-ink-faint" />
                    ) : (
                        <button
                            type="button"
                            aria-expanded={row.expanded}
                            aria-label={page.path}
                            className="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-raised hover:text-ink"
                            onClick={(event) => {
                                event.stopPropagation();
                                onToggle(page.id);
                            }}
                        >
                            <ChevronRightIcon
                                size={14}
                                className={cx("transition-transform duration-100", row.expanded && "rotate-90")}
                            />
                        </button>
                    )}
                    <span className="truncate font-mono text-xs text-ink">{page.path}</span>
                    <span className="truncate text-ink-dim">
                        {page.title === "" ? copy.pages.untitled : page.title}
                    </span>
                    {row.childCount === 0 ? null : (
                        <span className="shrink-0 font-mono text-2xs text-ink-faint">
                            {copy.pages.tree.children(row.childCount)}
                        </span>
                    )}
                </span>
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
        </TableRow>
    );
}

export interface PageTreeProps {
    siteId: string;
    query: PagesQuery;
    onQueryChange: (next: PagesQuery) => void;
    selectedId: string | null;
    index: EntityIndex;
    search: string;
    onPlan: () => void;
}

export function PageTree({
    siteId,
    query,
    onQueryChange,
    selectedId,
    index,
    search,
    onPlan,
}: PageTreeProps): ReactElement {
    const navigate = useNavigate();
    const overview = useSiteOverview(siteId);
    const total = overview.data?.pages.total;
    const [forced, setForced] = useState(false);
    const armed = forced || (total !== undefined && total <= treeFetchLimit);
    const tree = usePageTree(armed ? siteId : null);
    const roots = tree.data?.roots ?? null;

    const [expanded, setExpanded] = useState<ReadonlySet<string>>(() => new Set<string>());
    const [lifted, setLifted] = useState<ReadonlySet<string>>(() => new Set<string>());
    const seeded = useRef(false);

    useEffect(() => {
        if (roots === null || seeded.current) {
            return;
        }
        seeded.current = true;
        setExpanded(countNodes(roots) <= autoExpandLimit ? new Set(branchIds(roots)) : new Set<string>());
    }, [roots]);

    useEffect(() => {
        if (roots === null || selectedId === null) {
            return;
        }
        const trail = pathTo(roots, selectedId);
        if (trail.length === 0) {
            return;
        }
        setExpanded((held) => {
            if (trail.every((id) => held.has(id))) {
                return held;
            }
            const next = new Set(held);
            for (const id of trail) {
                next.add(id);
            }
            return next;
        });
    }, [roots, selectedId]);

    const nodeCount = useMemo(() => countNodes(roots), [roots]);
    const rows = useMemo(
        () => flattenTree(roots, expanded, lifted, childLimit),
        [roots, expanded, lifted],
    );

    const toTable = (): void => {
        onQueryChange({ ...query, view: "table" });
    };

    if (!armed) {
        return (
            <div className="min-h-0 flex-1 overflow-auto p-6">
                {overview.isPending ? (
                    <SkeletonRows rows={4} label={copy.pages.loading} className="max-w-lg" />
                ) : (
                    <Banner
                        tone={total === undefined ? "info" : "warn"}
                        title={
                            total === undefined
                                ? copy.pages.tree.unknownSize
                                : copy.pages.tree.guard(total)
                        }
                        body={copy.pages.tree.unbounded}
                        className="max-w-lg"
                        actions={
                            <>
                                <Button variant="primary" onClick={toTable}>
                                    {copy.pages.tree.useTable}
                                </Button>
                                <Button
                                    onClick={() => {
                                        setForced(true);
                                    }}
                                >
                                    {total === undefined ? copy.pages.tree.load : copy.pages.tree.loadAnyway}
                                </Button>
                            </>
                        }
                    />
                )}
            </div>
        );
    }

    if (tree.isPending) {
        return (
            <div className="min-h-0 flex-1 overflow-auto p-3">
                <SkeletonRows rows={12} label={copy.pages.loading} />
            </div>
        );
    }

    if (rows.length === 0) {
        return (
            <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-6">
                <EmptyState
                    icon={AccountTreeIcon}
                    title={copy.pages.empty.title}
                    body={copy.empty.pageTree}
                    actions={
                        <Button variant="primary" onClick={onPlan}>
                            {copy.pages.plan}
                        </Button>
                    }
                />
            </div>
        );
    }

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="flex h-7 shrink-0 items-center justify-between gap-2 border-b border-hairline px-3">
                <span className="font-mono text-2xs text-ink-faint">{copy.pages.tree.nodes(nodeCount)}</span>
                <div className="flex shrink-0 gap-1">
                    <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => {
                            setExpanded(new Set(branchIds(roots)));
                        }}
                    >
                        {copy.pages.tree.expandAll}
                    </Button>
                    <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => {
                            setExpanded(new Set<string>());
                            setLifted(new Set<string>());
                        }}
                    >
                        {copy.pages.tree.collapseAll}
                    </Button>
                </div>
            </div>
            <DenseTable columns={columns} label={copy.pages.tree.title} className="min-h-0 flex-1">
                <TableHead>
                    <div>{copy.pages.columns.path}</div>
                    <div>{copy.pages.columns.status}</div>
                    <div>{copy.pages.columns.entity}</div>
                    <div>{copy.pages.columns.drift}</div>
                    <div>{copy.pages.columns.synced}</div>
                </TableHead>
                <VirtualRows
                    count={rows.length}
                    rowHeight={rowHeight}
                    scrollKey={`${siteId}:tree`}
                    row={(position) => {
                        const row = rows[position];
                        if (row === undefined) {
                            return null;
                        }
                        if (row.kind === "more") {
                            return (
                                <div
                                    className="flex h-7 items-center border-b border-inset px-3"
                                    style={{ paddingLeft: `${12 + row.depth * indentStep}px` }}
                                >
                                    <Button
                                        size="sm"
                                        variant="ghost"
                                        onClick={() => {
                                            setLifted((held) => new Set(held).add(row.parentId));
                                        }}
                                    >
                                        {copy.pages.tree.showMore(row.hidden)}
                                    </Button>
                                </div>
                            );
                        }
                        return (
                            <BranchRow
                                row={row}
                                selected={row.page.id === selectedId}
                                entityName={
                                    row.page.entityId === null
                                        ? null
                                        : (index.byId.get(row.page.entityId)?.name ?? null)
                                }
                                onToggle={(pageId) => {
                                    setExpanded((held) => {
                                        const next = new Set(held);
                                        if (next.has(pageId)) {
                                            next.delete(pageId);
                                        } else {
                                            next.add(pageId);
                                        }
                                        return next;
                                    });
                                }}
                                onOpen={(pageId) => {
                                    void navigate(`/s/${siteId}/pages/${pageId}${search}`);
                                }}
                            />
                        );
                    }}
                />
            </DenseTable>
        </div>
    );
}
