import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useRuns } from "../../data/hooks/runs.js";
import type { Run } from "../../data/types.js";
import { absoluteTime, relativeTime, tokens, usd } from "../../domain/format.js";
import {
    Button,
    DenseTable,
    EmptyState,
    FilterAltIcon,
    HistoryIcon,
    PlayArrowIcon,
    ProgressBar,
    SkeletonRows,
    SortableHeader,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
} from "../../ui/index.js";
import { RunFilters } from "./filters.js";
import { pauseReasonShort, statusIcon, statusLabel, statusTone } from "./labels.js";
import { defaultQuery, filterOf, narrowed, nextSort, readQuery, searchOf, writeQuery } from "./params.js";
import type { RunsQuery } from "./params.js";
import { StartRunDialog } from "./start.js";
import { statusPaused } from "./statuses.js";
import { VirtualRows } from "../../ui/index.js";

const columns =
    "96px 116px minmax(120px,1.6fr) 92px minmax(90px,1fr) minmax(90px,1fr) minmax(90px,1fr)";
const rowHeight = 28;
const pageSize = 100;

interface RunRowProps {
    run: Run;
    onOpen: (runId: string) => void;
}

function RunRow({ run, onOpen }: RunRowProps): ReactElement {
    const cap = run.budget.maxUsd;
    return (
        <TableRow
            interactive={true}
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
                <StatusBadge tone={statusTone(run.status)} icon={statusIcon(run.status)}>
                    {statusLabel(run.status)}
                </StatusBadge>
            </TableCell>
            <TableCell>
                <ProgressBar
                    value={run.stats.done + run.stats.failed}
                    max={Math.max(run.stats.items, 1)}
                    tone={run.stats.failed > 0 ? "warn" : "accent"}
                    label={copy.runs.columns.progress}
                    leading={copy.runs.detail.items(run.stats.done, run.stats.items)}
                    trailing={
                        run.status === statusPaused && run.pauseReason !== ""
                            ? pauseReasonShort(run.pauseReason)
                            : tokens(run.stats.tokens)
                    }
                />
            </TableCell>
            <TableCell mono={true} align="right" muted={true}>
                {cap > 0 ? `${usd(run.stats.usd)} / ${usd(cap)}` : usd(run.stats.usd)}
            </TableCell>
            <TableCell mono={true} muted={true} title={absoluteTime(run.startedAt)}>
                {relativeTime(run.startedAt)}
            </TableCell>
            <TableCell mono={true} muted={true} title={absoluteTime(run.finishedAt)}>
                {relativeTime(run.finishedAt)}
            </TableCell>
            <TableCell mono={true} muted={true} title={absoluteTime(run.createdAt)}>
                {relativeTime(run.createdAt)}
            </TableCell>
        </TableRow>
    );
}

export function RunsScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const search = searchOf(query);
    const filter = useMemo(() => filterOf(siteId, query), [siteId, query]);
    const listed = useRuns(filter, query.sort, pageSize);
    const rows = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const [starting, setStarting] = useState(false);

    const change = (next: RunsQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const open = (runId: string): void => {
        void navigate(`/s/${siteId}/runs/${runId}`);
    };

    const toggle = (field: "createdAt" | "status"): void => {
        change({ ...query, sort: nextSort(query.sort, field) });
    };

    return (
        <div className="flex h-full min-h-0">
            <RunFilters query={query} onChange={change} />
            <div className="flex min-w-0 flex-1 flex-col">
                <header className="flex h-8 shrink-0 items-center justify-between gap-3 border-b border-hairline px-3">
                    <div className="flex min-w-0 items-center gap-2">
                        <h1 className="truncate text-xs font-semibold text-ink">{copy.runs.title}</h1>
                        <span className="truncate text-2xs text-ink-faint">{copy.runs.subtitle}</span>
                    </div>
                    <Button
                        size="sm"
                        variant="primary"
                        icon={PlayArrowIcon}
                        onClick={() => {
                            setStarting(true);
                        }}
                    >
                        {copy.runs.start.open}
                    </Button>
                </header>
                {listed.isPending ? (
                    <div className="min-h-0 flex-1 overflow-auto p-3">
                        <SkeletonRows rows={10} label={copy.runs.loading} />
                    </div>
                ) : rows.length === 0 ? (
                    <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-6">
                        {narrowed(query) ? (
                            <EmptyState
                                icon={FilterAltIcon}
                                title={copy.runs.noMatch}
                                body={copy.runs.noMatchBody}
                                actions={
                                    <Button
                                        onClick={() => {
                                            change({ ...defaultQuery, sort: query.sort });
                                        }}
                                    >
                                        {copy.runs.filters.reset}
                                    </Button>
                                }
                            />
                        ) : (
                            <EmptyState
                                icon={HistoryIcon}
                                title={copy.nav.runs}
                                body={copy.empty.runs}
                                actions={
                                    <Button
                                        variant="primary"
                                        onClick={() => {
                                            setStarting(true);
                                        }}
                                    >
                                        {copy.runs.start.open}
                                    </Button>
                                }
                            />
                        )}
                    </div>
                ) : (
                    <DenseTable columns={columns} label={copy.runs.title} className="min-h-0 flex-1">
                        <TableHead>
                            <div>{copy.runs.columns.kind}</div>
                            <SortableHeader
                                active={query.sort?.field === "status"}
                                direction={query.sort?.desc === true ? "desc" : "asc"}
                                onToggle={() => {
                                    toggle("status");
                                }}
                            >
                                {copy.runs.columns.status}
                            </SortableHeader>
                            <div>{copy.runs.columns.progress}</div>
                            <div className="text-right">{copy.runs.columns.spend}</div>
                            <div>{copy.runs.columns.started}</div>
                            <div>{copy.runs.columns.finished}</div>
                            <SortableHeader
                                active={query.sort?.field === "createdAt"}
                                direction={query.sort?.desc === true ? "desc" : "asc"}
                                onToggle={() => {
                                    toggle("createdAt");
                                }}
                            >
                                {copy.runs.columns.created}
                            </SortableHeader>
                        </TableHead>
                        <VirtualRows
                            count={rows.length}
                            rowHeight={rowHeight}
                            scrollKey={`${siteId}:runs${search}`}
                            row={(position) => {
                                const run = rows[position];
                                return run === undefined ? null : <RunRow run={run} onOpen={open} />;
                            }}
                            footer={
                                listed.hasNextPage ? (
                                    <div className="flex justify-center border-t border-hairline p-2">
                                        <Button
                                            size="sm"
                                            busy={listed.isFetchingNextPage}
                                            onClick={() => {
                                                void listed.fetchNextPage();
                                            }}
                                        >
                                            {copy.runs.loadMore}
                                        </Button>
                                    </div>
                                ) : null
                            }
                        />
                    </DenseTable>
                )}
            </div>
            <StartRunDialog
                open={starting}
                onOpenChange={setStarting}
                siteId={siteId}
                onStarted={(runId) => {
                    setStarting(false);
                    void navigate(`/s/${siteId}/runs/${runId}`);
                }}
            />
        </div>
    );
}
