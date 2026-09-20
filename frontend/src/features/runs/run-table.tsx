import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { Run } from "../../data/types.js";
import { absoluteTime, duration, relativeTime, usd } from "../../domain/format.js";
import {
    DenseTable,
    ProgressBar,
    SortableHeader,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
    VirtualRows,
} from "../../ui/index.js";
import type { RunSort } from "../../data/sorts.js";
import { pauseReasonShort, pauseReasonTone, statusIcon, statusLabel, statusTone } from "./labels.js";
import { spanMs } from "./span.js";
import { statusPaused } from "./statuses.js";

const columns = "84px 132px minmax(140px,1.8fr) 84px 88px 72px";
const rowHeight = 28;

interface RunRowProps {
    run: Run;
    now: number;
    selected: boolean;
    onOpen: (runId: string) => void;
}

function RunRow({ run, now, selected, onOpen }: RunRowProps): ReactElement {
    const cap = run.budget.maxUsd;
    const paused = run.status === statusPaused && run.pauseReason !== "";
    const lasted = spanMs(run.startedAt, run.finishedAt, now);
    return (
        <TableRow
            data-run-row={true}
            data-run-id={run.id}
            data-run-status={run.status}
            interactive={true}
            selected={selected}
            tabIndex={0}
            onClick={() => {
                onOpen(run.id);
            }}
            onKeyDown={(event) => {
                if (event.key === "Enter") {
                    event.preventDefault();
                    onOpen(run.id);
                }
            }}
        >
            <TableCell mono={true}>{run.kind}</TableCell>
            <TableCell>
                <span className="flex min-w-0 items-center gap-1">
                    <StatusBadge tone={statusTone(run.status)} icon={statusIcon(run.status)}>
                        {statusLabel(run.status)}
                    </StatusBadge>
                    {paused ? (
                        <StatusBadge tone={pauseReasonTone(run.pauseReason)} dot={false}>
                            {pauseReasonShort(run.pauseReason)}
                        </StatusBadge>
                    ) : null}
                </span>
            </TableCell>
            <TableCell>
                <ProgressBar
                    value={run.stats.done + run.stats.failed}
                    max={Math.max(run.stats.items, 1)}
                    tone={run.stats.failed > 0 ? "warn" : "accent"}
                    label={copy.runs.columns.progress}
                    leading={copy.runs.detail.items(run.stats.done, run.stats.items)}
                    trailing={run.stats.failed === 0 ? "" : `${String(run.stats.failed)} ${copy.runs.detail.failed}`}
                />
            </TableCell>
            <TableCell mono={true} align="right" muted={true}>
                {cap > 0 ? `${usd(run.stats.usd)} / ${usd(cap)}` : usd(run.stats.usd)}
            </TableCell>
            <TableCell mono={true} muted={true} title={absoluteTime(run.startedAt)}>
                {relativeTime(run.startedAt)}
            </TableCell>
            <TableCell mono={true} align="right" muted={true}>
                {lasted === null ? "" : duration(lasted)}
            </TableCell>
        </TableRow>
    );
}

export interface RunTableProps {
    rows: readonly Run[];
    now: number;
    sort: RunSort | null;
    selectedId: string | null;
    scrollKey: string;
    footer: ReactElement | null;
    onSortChange: (field: RunSort["field"]) => void;
    onOpen: (runId: string) => void;
}

export function RunTable({
    rows,
    now,
    sort,
    selectedId,
    scrollKey,
    footer,
    onSortChange,
    onOpen,
}: RunTableProps): ReactElement {
    return (
        <DenseTable columns={columns} label={copy.runs.title} className="min-h-0 flex-1">
            <TableHead>
                <div>{copy.runs.columns.kind}</div>
                <SortableHeader
                    active={sort?.field === "status"}
                    direction={sort?.desc === true ? "desc" : "asc"}
                    onToggle={() => {
                        onSortChange("status");
                    }}
                >
                    {copy.runs.columns.status}
                </SortableHeader>
                <div>{copy.runs.columns.progress}</div>
                <div className="text-right">{copy.runs.columns.spend}</div>
                <SortableHeader
                    active={sort?.field === "createdAt"}
                    direction={sort?.desc === true ? "desc" : "asc"}
                    onToggle={() => {
                        onSortChange("createdAt");
                    }}
                >
                    {copy.runs.columns.started}
                </SortableHeader>
                <div className="text-right">{copy.runs.columns.lasted}</div>
            </TableHead>
            <VirtualRows
                count={rows.length}
                rowHeight={rowHeight}
                scrollKey={scrollKey}
                row={(position) => {
                    const run = rows[position];
                    if (run === undefined) {
                        return null;
                    }
                    return (
                        <RunRow
                            run={run}
                            now={now}
                            selected={run.id === selectedId}
                            onOpen={onOpen}
                        />
                    );
                }}
                footer={footer}
            />
        </DenseTable>
    );
}
