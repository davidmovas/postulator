import type { ReactElement, ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import type { SiteMapNode } from "../../data/types.js";
import { Banner, Button, Skeleton, SkeletonRows, TableHead, TableRow, VirtualRows } from "../../ui/index.js";
import type { TickState } from "../../ui/selection.js";
import { MapTable, useMapColumns } from "./layout.js";
import type { Loaded } from "./loader.js";
import { BranchLoader } from "./loader.js";
import { flattenTree, openBranches, toggled } from "./model/tree.js";
import type { Branches, TreeRow } from "./model/tree.js";
import { indentStep, mapRowHeight, NodeRow } from "./row.js";

const said = copy.map;

export interface MapEntry {
    id: string;
    node: SiteMapNode;
    ancestors: readonly string[];
}

export interface MapScope {
    visible: readonly MapEntry[];
    branches: Branches;
}

export interface MapSelection {
    label: (node: SiteMapNode) => string;
    stateOf: (node: SiteMapNode, ancestors: readonly string[]) => TickState;
    onTick: (node: SiteMapNode, ancestors: readonly string[], ranged: boolean, scope: MapScope) => void;
}

export function MapHead(): ReactElement {
    const columns = useMapColumns();
    return (
        <TableHead>
            {columns.map((column) => (
                <div key={column} className="truncate">
                    {column === "under" ? null : said.columns[column]}
                </div>
            ))}
        </TableHead>
    );
}

export interface MapTreeProps {
    siteId: string;
    root: string;
    empty: ReactNode;
    within?: string;
    selection?: MapSelection;
    scrollKey?: string;
    expanded?: ReadonlySet<string>;
    onExpanded?: (next: ReadonlySet<string>) => void;
    onVisible?: (entries: readonly MapEntry[]) => void;
    onOpen: (pageId: string) => void;
    onUnder: (pageId: string) => void;
}

export function MapTree({
    siteId,
    root,
    empty,
    within,
    selection,
    scrollKey,
    expanded: shared,
    onExpanded,
    onVisible,
    onOpen,
    onUnder,
}: MapTreeProps): ReactElement {
    const [own, setOwn] = useState<ReadonlySet<string>>(new Set());
    const expanded = shared ?? own;
    const setExpanded = (change: (held: ReadonlySet<string>) => ReadonlySet<string>): void => {
        if (onExpanded === undefined) {
            setOwn(change);
        } else {
            onExpanded(change(expanded));
        }
    };
    const [loaded, setLoaded] = useState<ReadonlyMap<string, Loaded>>(new Map());

    const onLoaded = useCallback((parentId: string, held: Loaded) => {
        setLoaded((prev) => (prev.get(parentId) === held ? prev : new Map(prev).set(parentId, held)));
    }, []);

    const branches = useMemo<Branches>(() => new Map([...loaded].map(([key, held]) => [key, held.branch])), [loaded]);
    const open = useMemo(() => openBranches(root, expanded, branches), [root, expanded, branches]);
    const rows = useMemo(() => flattenTree(root, expanded, branches), [root, expanded, branches]);
    const scope = useMemo<MapScope>(
        () => ({
            visible: rows.flatMap((row) => (row.kind === "node" ? [{ id: row.node.page.pageId, node: row.node, ancestors: row.ancestors }] : [])),
            branches,
        }),
        [rows, branches],
    );
    const top = loaded.get(root);
    const failure = top === undefined || top.error === null ? null : react(top.error);

    useEffect(() => {
        onVisible?.(scope.visible);
    }, [scope.visible, onVisible]);

    const loaders = open.map((parentId) => (
        <BranchLoader key={parentId} siteId={siteId} parentId={parentId} within={within} onLoaded={onLoaded} />
    ));

    if (failure !== null && failure.kind !== "silent" && failure.kind !== "unlock") {
        return (
            <div className="p-4">
                {loaders}
                <Banner tone="danger" title={failure.message} />
            </div>
        );
    }
    if (top === undefined || (top.branch.pending && top.branch.nodes.length === 0)) {
        return (
            <div className="p-4">
                {loaders}
                <SkeletonRows rows={10} label={said.loading} />
            </div>
        );
    }
    if (top.branch.nodes.length === 0) {
        return (
            <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-6">
                {loaders}
                {empty}
            </div>
        );
    }

    const renderRow = (row: TreeRow): ReactNode => {
        switch (row.kind) {
            case "node":
                return (
                    <NodeRow
                        siteId={siteId}
                        node={row.node}
                        depth={row.depth}
                        expanded={row.expanded}
                        tick={
                            selection === undefined
                                ? undefined
                                : {
                                      label: selection.label(row.node),
                                      state: selection.stateOf(row.node, row.ancestors),
                                      onChange: (ranged) => {
                                          selection.onTick(row.node, row.ancestors, ranged, scope);
                                      },
                                  }
                        }
                        onToggle={(pageId) => {
                            setExpanded((held) => toggled(held, pageId));
                        }}
                        onOpen={onOpen}
                        onUnder={onUnder}
                    />
                );
            case "more":
                return (
                    <TableRow>
                        <div style={{ paddingLeft: `${(row.depth + 1) * indentStep}px` }}>
                            <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => {
                                    loaded.get(row.parentId)?.more();
                                }}
                            >
                                {said.more}
                            </Button>
                        </div>
                    </TableRow>
                );
            default:
                return (
                    <TableRow aria-label={said.loadingBranch}>
                        <div style={{ paddingLeft: `${(row.depth + 1) * indentStep}px` }}>
                            <Skeleton width="40%" height={10} />
                        </div>
                    </TableRow>
                );
        }
    };

    return (
        <MapTable label={said.title}>
            {loaders}
            <MapHead />
            <VirtualRows
                count={rows.length}
                rowHeight={mapRowHeight}
                scrollKey={scrollKey ?? `${siteId}:map:${root}`}
                row={(index) => {
                    const row = rows[index];
                    return row === undefined ? null : renderRow(row);
                }}
            />
        </MapTable>
    );
}
