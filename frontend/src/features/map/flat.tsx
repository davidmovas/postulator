import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import type { SiteMapFilter } from "../../data/types.js";
import { Banner, Button, EmptyState, FilterAltIcon, SkeletonRows, VirtualRows } from "../../ui/index.js";
import { MapTable } from "./layout.js";
import { useBranch } from "./loader.js";
import { NodeRow, mapRowHeight } from "./row.js";
import { MapHead } from "./tree.js";

const said = copy.map;

export interface FlatListProps {
    siteId: string;
    filter: SiteMapFilter;
    onOpen: (pageId: string) => void;
    onUnder: (pageId: string) => void;
    onReset: () => void;
}

export function FlatList({ siteId, filter, onOpen, onUnder, onReset }: FlatListProps): ReactElement {
    const found = useBranch(filter);
    const nodes = found.branch.nodes;
    const failure = found.error === null ? null : react(found.error);

    if (failure !== null && failure.kind !== "silent" && failure.kind !== "unlock") {
        return (
            <div className="p-4">
                <Banner tone="danger" title={failure.message} />
            </div>
        );
    }
    if (found.branch.pending && nodes.length === 0) {
        return (
            <div className="p-4">
                <SkeletonRows rows={10} label={said.loading} />
            </div>
        );
    }
    if (nodes.length === 0) {
        return (
            <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-6">
                <EmptyState
                    icon={FilterAltIcon}
                    title={said.noMatch.title}
                    actions={<Button onClick={onReset}>{said.noMatch.reset}</Button>}
                />
            </div>
        );
    }
    return (
        <MapTable label={said.title}>
            <MapHead />
            <VirtualRows
                count={nodes.length}
                rowHeight={mapRowHeight}
                onReachEnd={found.branch.hasMore ? found.more : undefined}
                row={(index) => {
                    const node = nodes[index];
                    return node === undefined ? null : (
                        <NodeRow
                            siteId={siteId}
                            node={node}
                            depth={0}
                            expanded={false}
                            onToggle={onUnder}
                            onOpen={onOpen}
                            onUnder={onUnder}
                        />
                    );
                }}
            />
        </MapTable>
    );
}
