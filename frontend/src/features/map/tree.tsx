import type { ReactElement, ReactNode } from "react";
import { useCallback, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { Banner, Button, DenseTable, Skeleton, SkeletonRows, TableHead, TableRow, VirtualRows } from "../../ui/index.js";
import type { Loaded } from "./loader.js";
import { BranchLoader } from "./loader.js";
import { flattenTree, openBranches, toggled } from "./model/tree.js";
import type { Branches, TreeRow } from "./model/tree.js";
import { indentStep, mapColumns, mapRowHeight, NodeRow } from "./row.js";

const said = copy.map;

export function MapHead(): ReactElement {
    return (
        <TableHead>
            <div>{said.columns.page}</div>
            <div>{said.columns.type}</div>
            <div>{said.columns.parent}</div>
            <div>{said.columns.body}</div>
            <div>{said.columns.links}</div>
            <div>{said.columns.findings}</div>
            <div>{said.columns.children}</div>
            <div />
        </TableHead>
    );
}

export interface MapTreeProps {
    siteId: string;
    root: string;
    empty: ReactNode;
    onOpen: (pageId: string) => void;
    onUnder: (pageId: string) => void;
}

export function MapTree({ siteId, root, empty, onOpen, onUnder }: MapTreeProps): ReactElement {
    const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
    const [loaded, setLoaded] = useState<ReadonlyMap<string, Loaded>>(new Map());

    const onLoaded = useCallback((parentId: string, held: Loaded) => {
        setLoaded((prev) => (prev.get(parentId) === held ? prev : new Map(prev).set(parentId, held)));
    }, []);

    const branches = useMemo<Branches>(() => new Map([...loaded].map(([key, held]) => [key, held.branch])), [loaded]);
    const open = useMemo(() => openBranches(root, expanded, branches), [root, expanded, branches]);
    const rows = useMemo(() => flattenTree(root, expanded, branches), [root, expanded, branches]);
    const top = loaded.get(root);
    const failure = top === undefined || top.error === null ? null : react(top.error);

    const loaders = open.map((parentId) => <BranchLoader key={parentId} siteId={siteId} parentId={parentId} onLoaded={onLoaded} />);

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
        <DenseTable columns={mapColumns} label={said.title} className="min-h-0 flex-1">
            {loaders}
            <MapHead />
            <VirtualRows
                count={rows.length}
                rowHeight={mapRowHeight}
                scrollKey={`${siteId}:map:${root}`}
                row={(index) => {
                    const row = rows[index];
                    return row === undefined ? null : renderRow(row);
                }}
            />
        </DenseTable>
    );
}
