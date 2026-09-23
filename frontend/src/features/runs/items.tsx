import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../copy/index.js";
import type { ItemProgress } from "../../data/runs/derive.js";
import type { RunItem } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { itemStatuses } from "../../generated/vocab.js";
import {
    Button,
    DenseTable,
    EmptyState,
    PendingActionsIcon,
    Segmented,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
} from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { itemViews } from "./authority.js";
import { itemBadge, itemNote } from "./hold.js";
import { statusLabel } from "./labels.js";
import type { RetryNotice } from "./log-view.js";
import type { PageIndex } from "./page-index.js";
import { pathOf } from "./page-index.js";
import { StepCell } from "./step-cell.js";
import { VirtualRows } from "../../ui/index.js";

const columns = "minmax(110px,2fr) 104px minmax(110px,2.2fr) 36px minmax(80px,1.6fr) 72px";
const rowHeight = 28;

export interface ItemStatusTabsProps {
    value: string;
    onChange: (next: string) => void;
}

const itemStatusOptions: readonly SegmentedOption<string>[] = [
    { value: "", label: copy.runs.filters.anyItemStatus },
    ...itemStatuses.map((status) => ({ value: status, label: statusLabel(status) })),
];

export function ItemStatusTabs({ value, onChange }: ItemStatusTabsProps): ReactElement {
    return (
        <Segmented
            label={copy.runs.filters.anyItemStatus}
            value={value}
            options={itemStatusOptions}
            onValueChange={onChange}
            size="sm"
        />
    );
}

export interface RunItemTableProps {
    runId: string;
    items: readonly RunItem[];
    progress: ReadonlyMap<string, ItemProgress>;
    retries: ReadonlyMap<string, RetryNotice>;
    terminal: boolean;
    steps: readonly string[];
    index: PageIndex;
    selectedId: string | null;
    now: number;
    narrowed: boolean;
    hasMore: boolean;
    loadingMore: boolean;
    onLoadMore: () => void;
    onOpen: (itemId: string) => void;
}

export function RunItemTable({
    runId,
    items,
    progress,
    retries,
    terminal,
    steps,
    index,
    selectedId,
    now,
    narrowed,
    hasMore,
    loadingMore,
    onLoadMore,
    onOpen,
}: RunItemTableProps): ReactElement {
    const views = useMemo(() => itemViews(items, progress, terminal), [items, progress, terminal]);

    if (views.length === 0) {
        return (
            <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-6">
                <EmptyState
                    icon={PendingActionsIcon}
                    title={copy.nav.runs}
                    body={narrowed ? copy.runs.noItemMatch : copy.empty.runItems}
                />
            </div>
        );
    }

    return (
        <DenseTable columns={columns} label={copy.runs.columns.path} className="min-h-0 flex-1">
            <TableHead>
                <div>{copy.runs.columns.path}</div>
                <div>{copy.runs.columns.status}</div>
                <div>{copy.runs.columns.step}</div>
                <div className="text-right">{copy.runs.columns.attempts}</div>
                <div>{copy.runs.columns.error}</div>
                <div>{copy.runs.columns.updated}</div>
            </TableHead>
            <VirtualRows
                count={views.length}
                rowHeight={rowHeight}
                scrollKey={`${runId}:items`}
                row={(position) => {
                    const view = views[position];
                    if (view === undefined) {
                        return null;
                    }
                    const item = view.item;
                    const badge = itemBadge(item);
                    const note = itemNote(item);
                    const failure = item.error === "" ? view.stepFailure?.code : item.error;
                    return (
                        <TableRow
                            data-item-row={true}
                            data-item-id={item.id}
                            data-item-status={item.status}
                            interactive={true}
                            selected={item.id === selectedId}
                            tabIndex={0}
                            onClick={() => {
                                onOpen(item.id);
                            }}
                            onKeyDown={(event) => {
                                if (event.key === "Enter") {
                                    event.preventDefault();
                                    onOpen(item.id);
                                }
                            }}
                        >
                            <TableCell mono={true} title={item.targetId}>
                                {pathOf(index, item.targetId)}
                            </TableCell>
                            <TableCell>
                                <StatusBadge tone={badge.tone} icon={badge.icon}>
                                    {badge.label}
                                </StatusBadge>
                            </TableCell>
                            <TableCell>
                                <StepCell
                                    view={view}
                                    steps={steps}
                                    retry={retries.get(item.id)}
                                    now={now}
                                />
                            </TableCell>
                            <TableCell mono={true} align="right" muted={true}>
                                {item.attempts}
                            </TableCell>
                            <TableCell
                                muted={true}
                                title={
                                    item.error !== ""
                                        ? item.error
                                        : note !== ""
                                          ? item.note
                                          : (view.stepFailure?.message ?? "")
                                }
                            >
                                {item.error !== "" ? (
                                    <span className="text-danger">{failure}</span>
                                ) : note !== "" ? (
                                    note
                                ) : (
                                    (failure ?? "")
                                )}
                            </TableCell>
                            <TableCell mono={true} muted={true} title={absoluteTime(item.updatedAt)}>
                                {relativeTime(item.updatedAt)}
                            </TableCell>
                        </TableRow>
                    );
                }}
                footer={
                    hasMore ? (
                        <div className="flex justify-center border-t border-hairline p-2">
                            <Button size="sm" busy={loadingMore} onClick={onLoadMore}>
                                {copy.app.loadMore}
                            </Button>
                        </div>
                    ) : null
                }
            />
        </DenseTable>
    );
}
