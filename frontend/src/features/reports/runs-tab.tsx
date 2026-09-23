import type { ReactElement } from "react";
import { useMemo } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { react } from "../../data/errors.js";
import { useRuns } from "../../data/hooks/runs.js";
import { absoluteTime, relativeTime, usd } from "../../domain/format.js";
import {
    Banner,
    Button,
    DenseTable,
    EmptyState,
    HistoryIcon,
    SkeletonRows,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
} from "../../ui/index.js";
import { kindLabel, statusLabel, statusTone } from "../runs/labels.js";
import { finished } from "./model/runs.js";

const grid = "minmax(11rem,1.4fr) minmax(6rem,1fr) 4rem 7rem 5rem";
const pageSize = 100;

export interface RunsTabProps {
    siteId: string;
    selectedId: string;
    onSelect: (runId: string) => void;
}

export function RunsTab({ siteId, selectedId, onSelect }: RunsTabProps): ReactElement {
    const navigate = useNavigate();
    const listed = useRuns({ siteId }, null, pageSize);
    const rows = useMemo(() => finished(flatten(listed.data?.pages)), [listed.data]);

    if (listed.isPending) {
        return (
            <div className="p-4">
                <SkeletonRows rows={10} label={copy.runs.loading} />
            </div>
        );
    }

    const failure = listed.error === null ? null : react(listed.error);
    if (failure !== null && failure.kind !== "silent" && failure.kind !== "unlock") {
        return (
            <div className="p-4">
                <Banner
                    tone="danger"
                    title={failure.message}
                    actions={
                        <Button size="sm" variant="secondary" onClick={() => void listed.refetch()}>
                            {copy.app.retry}
                        </Button>
                    }
                />
            </div>
        );
    }

    if (rows.length === 0) {
        return (
            <div className="p-4">
                <EmptyState
                    icon={HistoryIcon}
                    title={copy.reports.runs.empty}
                    actions={
                        <Button
                            variant="primary"
                            onClick={() => {
                                void navigate(`/s/${siteId}/runs?action=new`);
                            }}
                        >
                            {copy.reports.empty.startRun}
                        </Button>
                    }
                />
            </div>
        );
    }

    return (
        <div className="min-h-0 flex-1 overflow-auto">
            <DenseTable columns={grid} label={copy.reports.tabs.runs}>
                <TableHead>
                    <span>{copy.reports.runs.kind}</span>
                    <span>{copy.reports.runs.when}</span>
                    <span>{copy.reports.runs.items}</span>
                    <span>{copy.reports.runs.outcome}</span>
                    <span>{copy.reports.runs.cost}</span>
                </TableHead>
                {rows.map((run) => (
                    <TableRow
                        key={run.id}
                        interactive={true}
                        selected={run.id === selectedId}
                        data-run-id={run.id}
                        onClick={() => {
                            onSelect(run.id);
                        }}
                    >
                        <TableCell>
                            <span className="flex min-w-0 items-center gap-2">
                                <StatusBadge tone={statusTone(run.status)}>{statusLabel(run.status)}</StatusBadge>
                                <span className="truncate">{kindLabel(run.kind)}</span>
                            </span>
                        </TableCell>
                        <TableCell muted={true} title={absoluteTime(run.finishedAt ?? run.createdAt)}>
                            {relativeTime(run.finishedAt ?? run.createdAt)}
                        </TableCell>
                        <TableCell mono={true} align="right">
                            {run.stats.items}
                        </TableCell>
                        <TableCell mono={true}>
                            <span className={run.stats.failed > 0 ? "text-danger" : "text-ink"}>
                                {run.stats.done} / {run.stats.failed}
                            </span>
                        </TableCell>
                        <TableCell mono={true} align="right">
                            {usd(run.stats.usd)}
                        </TableCell>
                    </TableRow>
                ))}
            </DenseTable>
            {listed.hasNextPage !== true ? null : (
                <div className="p-3">
                    <Button
                        size="sm"
                        variant="secondary"
                        busy={listed.isFetchingNextPage}
                        onClick={() => void listed.fetchNextPage()}
                    >
                        {copy.app.loadMore}
                    </Button>
                </div>
            )}
        </div>
    );
}
