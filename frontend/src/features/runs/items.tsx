import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../copy/index.js";
import type { ItemProgress } from "../../data/runs/derive.js";
import type { RunItem } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { itemStatuses } from "../../generated/vocab.js";
import {
    Button,
    cx,
    DenseTable,
    EmptyState,
    PendingActionsIcon,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
} from "../../ui/index.js";
import { itemViews } from "./authority.js";
import { statusIcon, statusLabel, statusTone } from "./labels.js";
import type { RetryNotice } from "./log-view.js";
import type { PageIndex } from "./page-index.js";
import { pathOf } from "./page-index.js";
import { StepCell } from "./step-cell.js";
import { VirtualRows } from "../../ui/index.js";

const columns = "minmax(140px,2fr) 108px minmax(160px,2.2fr) 44px minmax(120px,1.6fr) 96px";
const rowHeight = 28;

export interface ItemStatusTabsProps {
    value: string;
    onChange: (next: string) => void;
}

export function ItemStatusTabs({ value, onChange }: ItemStatusTabsProps): ReactElement {
    return (
        <div className="flex shrink-0 flex-wrap items-center gap-0.5">
            <button
                type="button"
                aria-pressed={value === ""}
                onClick={() => {
                    onChange("");
                }}
                className={cx(
                    "h-5 rounded-sm px-1.5 text-2xs font-medium transition-colors duration-100",
                    value === "" ? "bg-raised text-ink" : "text-ink-dim hover:bg-inset hover:text-ink",
                )}
            >
                {copy.runs.filters.anyItemStatus}
            </button>
            {itemStatuses.map((status) => (
                <button
                    key={status}
                    type="button"
                    aria-pressed={value === status}
                    onClick={() => {
                        onChange(value === status ? "" : status);
                    }}
                    className={cx(
                        "h-5 rounded-sm px-1.5 text-2xs font-medium transition-colors duration-100",
                        value === status ? "bg-raised text-ink" : "text-ink-dim hover:bg-inset hover:text-ink",
                    )}
                >
                    {statusLabel(status)}
                </button>
            ))}
        </div>
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
                    const failure = item.error === "" ? view.stepFailure?.code : item.error;
                    return (
                        <TableRow
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
                                <StatusBadge tone={statusTone(item.status)} icon={statusIcon(item.status)}>
                                    {statusLabel(item.status)}
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
                                title={item.error === "" ? (view.stepFailure?.message ?? "") : item.error}
                            >
                                {item.error === "" ? (
                                    (failure ?? "")
                                ) : (
                                    <span className="text-danger">{failure}</span>
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
                                {copy.runs.loadMore}
                            </Button>
                        </div>
                    ) : null
                }
            />
        </DenseTable>
    );
}
