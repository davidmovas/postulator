import type { ReactElement, ReactNode } from "react";
import { Fragment, useCallback, useEffect, useMemo, useState } from "react";

import { copy } from "../../../copy/index.js";
import { react } from "../../../data/errors.js";
import { useAddressTree } from "../../../data/hooks/pages.js";
import type { AddressNode } from "../../../data/types.js";
import {
    Banner,
    Button,
    CenterFocusWeakIcon,
    Checkbox,
    ChevronRightIcon,
    cx,
    FolderIcon,
    IconButton,
    shiftHeld,
    Skeleton,
    SkeletonRows,
    TableCell,
    TableRow,
    VirtualRows,
} from "../../../ui/index.js";
import type { TickState } from "../../../ui/selection.js";
import { MapTable, useMapColumns } from "../../map/layout.js";
import type { MapColumn } from "../../map/model/columns.js";
import { indentStep, mapRowHeight, NodeRow } from "../../map/row.js";
import type { NodeTick } from "../../map/row.js";
import { MapHead } from "../../map/tree.js";
import type { AddressBranch, AddressBranches, AddressRow } from "./model.js";
import { asMapNode, flattenAddress, keyOf, openFolders } from "./model.js";

const said = copy.pages.tree;

export interface AddressEntry {
    key: string;
    node: AddressNode;
    ancestors: readonly string[];
}

export interface AddressScope {
    visible: readonly AddressEntry[];
    branches: AddressBranches;
}

export interface AddressSelection {
    label: (node: AddressNode) => string;
    stateOf: (node: AddressNode, ancestors: readonly string[]) => TickState;
    onTick: (node: AddressNode, ancestors: readonly string[], ranged: boolean, scope: AddressScope) => void;
}

interface Loaded {
    branch: AddressBranch;
    error: unknown;
    more: () => void;
}

interface FolderLoaderProps {
    siteId: string;
    folder: string;
    within?: string;
    onLoaded: (folder: string, loaded: Loaded) => void;
}

function FolderLoader({ siteId, folder, within, onLoaded }: FolderLoaderProps): null {
    const listed = useAddressTree({
        siteId,
        ...(folder === "" ? {} : { folder }),
        ...(within === undefined || within === "" ? {} : { within }),
    });
    const pages = listed.data?.pages;
    const nodes = useMemo<readonly AddressNode[]>(() => (pages ?? []).flatMap((view) => view.nodes.items ?? []), [pages]);
    const { hasNextPage, isPending, isError, isFetchingNextPage, fetchNextPage, error } = listed;
    const loaded = useMemo<Loaded>(
        () => ({
            branch: { nodes, hasMore: hasNextPage, pending: isPending && !isError },
            error,
            more: () => {
                if (!isFetchingNextPage) {
                    void fetchNextPage();
                }
            },
        }),
        [nodes, hasNextPage, isPending, isError, error, isFetchingNextPage, fetchNextPage],
    );

    useEffect(() => {
        onLoaded(folder, loaded);
    }, [folder, loaded, onLoaded]);

    return null;
}

interface FolderRowProps {
    node: AddressNode;
    depth: number;
    expanded: boolean;
    tick?: NodeTick;
    onToggle: () => void;
    onUnder: () => void;
}

function FolderRow({ node, depth, expanded, tick, onToggle, onUnder }: FolderRowProps): ReactElement {
    const columns = useMapColumns();
    const cellOf = (column: MapColumn): ReactElement => {
        switch (column) {
            case "page":
                return (
                    <TableCell>
                        <span className="flex min-w-0 items-center gap-1.5" style={{ paddingLeft: `${depth * indentStep}px` }}>
                            {tick === undefined ? null : (
                                <span
                                    className="flex shrink-0 items-center"
                                    onMouseDown={(event) => {
                                        if (event.shiftKey) {
                                            event.preventDefault();
                                        }
                                    }}
                                >
                                    <Checkbox
                                        aria-label={tick.label}
                                        checked={tick.state === "on"}
                                        indeterminate={tick.state === "some"}
                                        onChange={(event) => {
                                            tick.onChange(shiftHeld(event));
                                        }}
                                    />
                                </span>
                            )}
                            <button
                                type="button"
                                aria-expanded={expanded}
                                aria-label={expanded ? copy.map.collapse(node.path) : copy.map.expand(node.path)}
                                className="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-raised hover:text-ink"
                                onClick={onToggle}
                            >
                                <ChevronRightIcon size={14} className={cx("transition-transform duration-100", expanded && "rotate-90")} />
                            </button>
                            <span title={said.folder} className="flex min-w-0 items-center gap-1.5 text-ink-dim">
                                <FolderIcon size={13} className="shrink-0 text-ink-faint" />
                                <span className="truncate font-mono text-xs">{node.path}</span>
                            </span>
                        </span>
                    </TableCell>
                );
            case "children":
                return (
                    <TableCell mono={true} muted={true} title={said.pages(node.pages)}>
                        {node.children}
                    </TableCell>
                );
            case "under":
                return (
                    <TableCell>
                        <IconButton size="sm" variant="ghost" icon={CenterFocusWeakIcon} label={said.under(node.path)} onClick={onUnder} />
                    </TableCell>
                );
            default:
                return <TableCell muted={true}>{null}</TableCell>;
        }
    };
    return (
        <TableRow data-address-folder={node.path} data-map-tick={tick?.state}>
            {columns.map((column) => (
                <Fragment key={column}>{cellOf(column)}</Fragment>
            ))}
        </TableRow>
    );
}

export interface AddressTreeProps {
    siteId: string;
    empty: ReactNode;
    within?: string;
    selection?: AddressSelection;
    scrollKey: string;
    expanded: ReadonlySet<string>;
    onExpanded: (next: ReadonlySet<string>) => void;
    onVisible?: (entries: readonly AddressEntry[]) => void;
    onOpen: (pageId: string) => void;
    onUnder: (path: string) => void;
}

function toggledFolder(expanded: ReadonlySet<string>, path: string): ReadonlySet<string> {
    const next = new Set(expanded);
    if (!next.delete(path)) {
        next.add(path);
    }
    return next;
}

export function AddressTree({
    siteId,
    empty,
    within,
    selection,
    scrollKey,
    expanded,
    onExpanded,
    onVisible,
    onOpen,
    onUnder,
}: AddressTreeProps): ReactElement {
    const [loaded, setLoaded] = useState<ReadonlyMap<string, Loaded>>(new Map());

    const onLoaded = useCallback((folder: string, held: Loaded) => {
        setLoaded((prev) => (prev.get(folder) === held ? prev : new Map(prev).set(folder, held)));
    }, []);

    const branches = useMemo<AddressBranches>(() => new Map([...loaded].map(([key, held]) => [key, held.branch])), [loaded]);
    const open = useMemo(() => openFolders(expanded, branches), [expanded, branches]);
    const rows = useMemo(() => flattenAddress(expanded, branches), [expanded, branches]);
    const scope = useMemo<AddressScope>(
        () => ({
            visible: rows.flatMap((row) => (row.kind === "node" ? [{ key: keyOf(row.node), node: row.node, ancestors: row.ancestors }] : [])),
            branches,
        }),
        [rows, branches],
    );
    const top = loaded.get("");
    const failure = top === undefined || top.error === null ? null : react(top.error);

    useEffect(() => {
        onVisible?.(scope.visible);
    }, [scope.visible, onVisible]);

    const loaders = open.map((folder) => <FolderLoader key={folder} siteId={siteId} folder={folder} within={within} onLoaded={onLoaded} />);

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

    const tickOf = (row: Extract<AddressRow, { kind: "node" }>): NodeTick | undefined =>
        selection === undefined
            ? undefined
            : {
                  label: selection.label(row.node),
                  state: selection.stateOf(row.node, row.ancestors),
                  onChange: (ranged) => {
                      selection.onTick(row.node, row.ancestors, ranged, scope);
                  },
              };

    const renderRow = (row: AddressRow): ReactNode => {
        switch (row.kind) {
            case "node": {
                const toggle = (): void => {
                    onExpanded(toggledFolder(expanded, row.node.path));
                };
                const drawn = asMapNode(row.node);
                if (drawn === null) {
                    return (
                        <FolderRow
                            node={row.node}
                            depth={row.depth}
                            expanded={row.expanded}
                            tick={tickOf(row)}
                            onToggle={toggle}
                            onUnder={() => {
                                onUnder(row.node.path);
                            }}
                        />
                    );
                }
                return (
                    <NodeRow
                        siteId={siteId}
                        node={drawn}
                        depth={row.depth}
                        expanded={row.expanded}
                        tick={tickOf(row)}
                        onToggle={toggle}
                        onOpen={onOpen}
                        onUnder={() => {
                            onUnder(row.node.path);
                        }}
                    />
                );
            }
            case "more":
                return (
                    <TableRow>
                        <div style={{ paddingLeft: `${(row.depth + 1) * indentStep}px` }}>
                            <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => {
                                    loaded.get(row.folder)?.more();
                                }}
                            >
                                {copy.map.more}
                            </Button>
                        </div>
                    </TableRow>
                );
            default:
                return (
                    <TableRow aria-label={copy.map.loadingBranch}>
                        <div style={{ paddingLeft: `${(row.depth + 1) * indentStep}px` }}>
                            <Skeleton width="40%" height={10} />
                        </div>
                    </TableRow>
                );
        }
    };

    return (
        <MapTable label={said.label}>
            {loaders}
            <MapHead />
            <VirtualRows
                count={rows.length}
                rowHeight={mapRowHeight}
                scrollKey={scrollKey}
                row={(index) => {
                    const row = rows[index];
                    return row === undefined ? null : renderRow(row);
                }}
            />
        </MapTable>
    );
}
