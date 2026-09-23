import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import type { Run, Schedule } from "../../data/types.js";
import { cadenceWords } from "../../domain/cron.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { DenseTable, StatusBadge, Switch, TableCell, TableHead, TableRow } from "../../ui/index.js";
import { statusLabel, statusTone } from "../runs/labels.js";
import { targetsSentence } from "./labels.js";

const grid = "minmax(6rem,1.3fr) minmax(5rem,1.1fr) minmax(5rem,1.2fr) 6.5rem 10rem 2.5rem";

export interface SchedulesTableProps {
    siteId: string;
    rows: readonly Schedule[];
    entityNames: ReadonlyMap<string, string>;
    lastRuns: ReadonlyMap<string, Run>;
    selectedId: string;
    onSelect: (id: string) => void;
    onToggle: (schedule: Schedule) => void;
}

export function SchedulesTable({
    siteId,
    rows,
    entityNames,
    lastRuns,
    selectedId,
    onSelect,
    onToggle,
}: SchedulesTableProps): ReactElement {
    return (
        <DenseTable columns={grid} label={copy.schedules.title}>
            <TableHead>
                <span>{copy.schedules.columns.name}</span>
                <span>{copy.schedules.columns.targets}</span>
                <span>{copy.schedules.columns.cadence}</span>
                <span>{copy.schedules.columns.next}</span>
                <span>{copy.schedules.columns.last}</span>
                <span>{copy.schedules.columns.enabled}</span>
            </TableHead>
            {rows.map((row) => {
                const lastRun = row.lastRunId === null || row.lastRunId === undefined ? undefined : lastRuns.get(row.lastRunId);
                return (
                    <TableRow
                        key={row.id}
                        interactive={true}
                        selected={row.id === selectedId}
                        data-schedule-id={row.id}
                        onClick={() => {
                            onSelect(row.id);
                        }}
                    >
                        <TableCell title={row.name}>
                            <span className={row.enabled ? "text-ink" : "text-ink-dim"}>{row.name}</span>
                        </TableCell>
                        <TableCell muted={true}>
                            {targetsSentence(
                                row.status ?? "",
                                row.limit,
                                row.entityId === null || row.entityId === undefined
                                    ? ""
                                    : (entityNames.get(row.entityId) ?? ""),
                            )}
                        </TableCell>
                        <TableCell muted={true} title={row.cron ?? ""}>
                            {cadenceWords(row.cron ?? "", row.intervalMinutes ?? 0)}
                        </TableCell>
                        <TableCell muted={true} title={absoluteTime(row.nextRunAt)}>
                            {row.enabled && row.nextRunAt !== null ? relativeTime(row.nextRunAt) : copy.schedules.row.off}
                        </TableCell>
                        <TableCell>
                            {row.lastRunId === null || row.lastRunId === undefined ? (
                                <span className="text-ink-faint">{copy.schedules.row.never}</span>
                            ) : (
                                <Link
                                    to={`/s/${siteId}/runs/${row.lastRunId}`}
                                    title={absoluteTime(lastRun?.finishedAt ?? lastRun?.createdAt ?? null)}
                                    className="flex min-w-0 items-center gap-1.5"
                                    onClick={(event) => {
                                        event.stopPropagation();
                                    }}
                                >
                                    {lastRun === undefined ? null : (
                                        <StatusBadge tone={statusTone(lastRun.status)}>
                                            {statusLabel(lastRun.status)}
                                        </StatusBadge>
                                    )}
                                    <span className="truncate text-2xs text-ink-faint">
                                        {relativeTime(lastRun?.finishedAt ?? lastRun?.createdAt ?? null)}
                                    </span>
                                </Link>
                            )}
                        </TableCell>
                        <TableCell>
                            <span
                                onClick={(event) => {
                                    event.stopPropagation();
                                }}
                            >
                                <Switch
                                    checked={row.enabled}
                                    aria-label={copy.schedules.row.toggle(row.name)}
                                    onChange={() => {
                                        onToggle(row);
                                    }}
                                />
                            </span>
                        </TableCell>
                    </TableRow>
                );
            })}
        </DenseTable>
    );
}
