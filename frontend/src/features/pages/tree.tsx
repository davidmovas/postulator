import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import {
    AccountTreeIcon,
    Banner,
    Button,
    DenseTable,
    EmptyState,
    SkeletonRows,
    TableHead,
    VirtualRows,
} from "../../ui/index.js";
import type { EntityIndex } from "./entities.js";
import { indentStep, PageTreeRow, treeColumns } from "./tree-row.js";
import type { TreeView } from "./tree-state.js";

const rowHeight = 28;

export interface PageTreeProps {
    view: TreeView;
    index: EntityIndex;
    selectedId: string | null;
    onSelect: (pageId: string) => void;
    onOpen: (pageId: string) => void;
    onCreate: () => void;
}

export function PageTree({ view, index, selectedId, onSelect, onOpen, onCreate }: PageTreeProps): ReactElement {
    if (!view.armed) {
        return (
            <div className="min-h-0 flex-1 overflow-auto p-4">
                {view.sizing ? (
                    <SkeletonRows rows={4} label={copy.pages.loading} className="max-w-lg" />
                ) : (
                    <Banner
                        tone={view.total === undefined ? "info" : "warn"}
                        title={
                            view.total === undefined
                                ? copy.pages.tree.unknownSize
                                : copy.pages.tree.guard(view.total)
                        }
                        body={copy.pages.tree.unbounded}
                        className="max-w-lg"
                        actions={
                            <Button variant="primary" onClick={view.arm}>
                                {view.total === undefined ? copy.pages.tree.load : copy.pages.tree.loadAnyway}
                            </Button>
                        }
                    />
                )}
            </div>
        );
    }

    if (view.pending) {
        return (
            <div className="min-h-0 flex-1 overflow-auto p-3">
                <SkeletonRows rows={12} label={copy.pages.loading} />
            </div>
        );
    }

    if (view.rows.length === 0) {
        return (
            <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-6">
                <EmptyState
                    icon={AccountTreeIcon}
                    title={copy.pages.empty.title}
                    body={copy.empty.pageTree}
                    actions={
                        <Button variant="primary" onClick={onCreate}>
                            {copy.pages.newPage}
                        </Button>
                    }
                />
            </div>
        );
    }

    return (
        <DenseTable columns={treeColumns} label={copy.pages.tree.title} className="min-h-0 flex-1">
            <TableHead>
                <div>{copy.pages.columns.path}</div>
                <div>{copy.pages.columns.status}</div>
                <div>{copy.pages.columns.entity}</div>
                <div>{copy.pages.columns.drift}</div>
                <div>{copy.pages.columns.synced}</div>
            </TableHead>
            <VirtualRows
                count={view.rows.length}
                rowHeight={rowHeight}
                scrollKey="pages:tree"
                row={(position) => {
                    const row = view.rows[position];
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
                                        view.lift(row.parentId);
                                    }}
                                >
                                    {copy.pages.tree.showMore(row.hidden)}
                                </Button>
                            </div>
                        );
                    }
                    return (
                        <PageTreeRow
                            row={row}
                            selected={row.page.id === selectedId}
                            entityName={
                                row.page.entityId === null
                                    ? null
                                    : (index.byId.get(row.page.entityId)?.name ?? null)
                            }
                            onToggle={view.toggle}
                            onSelect={onSelect}
                            onOpen={onOpen}
                        />
                    );
                }}
            />
        </DenseTable>
    );
}
