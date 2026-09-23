import type { ReactElement } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import type { Page } from "../../data/types.js";
import { isBrowsable, openExternal } from "../../data/host.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    Button,
    DenseTable,
    EmptyState,
    OpenInNewIcon,
    Panel,
    PanelHeader,
    StatusBadge,
    SyncIcon,
    TableCell,
    TableHead,
    TableRow,
    TaskAltIcon,
} from "../../ui/index.js";
import { liveUrl } from "../pages/summary.js";
import { kindLabel as entityKindLabel } from "../graph/labels.js";
import { reasonLabel, reasonTone } from "./labels.js";
import type { CoverageRow } from "./model/site.js";

const coverageGrid = "minmax(0,1.6fr) 7rem minmax(0,1.4fr) 9rem 6rem";
const driftGrid = "minmax(0,2fr) 9rem 9rem 12rem";

export interface CoverageTableProps {
    siteId: string;
    rows: readonly CoverageRow[];
}

export function CoverageTable({ siteId, rows }: CoverageTableProps): ReactElement {
    const navigate = useNavigate();
    return (
        <Panel>
            <PanelHeader title={copy.reports.coverage.title}>
                <span className="text-2xs text-ink-faint">{copy.reports.coverage.order}</span>
            </PanelHeader>
            {rows.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={TaskAltIcon} title={copy.empty.entities} />
                </div>
            ) : (
                <div className="max-h-96 overflow-auto">
                <DenseTable columns={coverageGrid} label={copy.reports.coverage.title}>
                    <TableHead>
                        <span>{copy.reports.coverage.entity}</span>
                        <span>{copy.reports.coverage.kind}</span>
                        <span>{copy.reports.coverage.path}</span>
                        <span>{copy.reports.coverage.state}</span>
                        <span>{copy.reports.coverage.links}</span>
                    </TableHead>
                    {rows.map((row) => (
                        <TableRow
                            key={row.entityId}
                            interactive={true}
                            data-entity-id={row.entityId}
                            onClick={() => {
                                void navigate(`/s/${siteId}/graph/${row.entityId}`);
                            }}
                        >
                            <TableCell title={row.name}>{row.name}</TableCell>
                            <TableCell muted={true}>{entityKindLabel(row.kind)}</TableCell>
                            <TableCell mono={true} muted={true} title={row.path}>
                                {row.path}
                            </TableCell>
                            <TableCell>
                                <StatusBadge tone={reasonTone(row.reason)}>{reasonLabel(row.reason)}</StatusBadge>
                            </TableCell>
                            <TableCell mono={true} align="right" muted={row.links === 0}>
                                {copy.reports.coverage.linkCount(row.links)}
                            </TableCell>
                        </TableRow>
                    ))}
                </DenseTable>
                </div>
            )}
        </Panel>
    );
}

export interface DriftTableProps {
    baseUrl: string;
    rows: readonly Page[];
    syncing: boolean;
    onResync: () => void;
}

export function DriftTable({ baseUrl, rows, syncing, onResync }: DriftTableProps): ReactElement {
    return (
        <Panel>
            <PanelHeader title={copy.reports.drift.title}>
                <Button
                    size="sm"
                    variant="secondary"
                    icon={SyncIcon}
                    busy={syncing}
                    title={copy.reports.drift.resyncTitle}
                    onClick={onResync}
                >
                    {copy.reports.drift.resync}
                </Button>
            </PanelHeader>
            {rows.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={TaskAltIcon} title={copy.reports.drift.none} />
                </div>
            ) : (
                <DenseTable columns={driftGrid} label={copy.reports.drift.title}>
                    <TableHead>
                        <span>{copy.reports.drift.path}</span>
                        <span>{copy.reports.drift.synced}</span>
                        <span>{copy.reports.drift.changed}</span>
                        <span />
                    </TableHead>
                    {rows.map((page) => {
                        const url = liveUrl(baseUrl, page.path);
                        return (
                            <TableRow key={page.id} data-page-id={page.id}>
                                <TableCell mono={true} title={page.path}>
                                    {page.path}
                                </TableCell>
                                <TableCell muted={true} title={absoluteTime(page.lastSyncedAt)}>
                                    {relativeTime(page.lastSyncedAt)}
                                </TableCell>
                                <TableCell title={absoluteTime(page.wpModifiedAt)}>
                                    <span className="text-warn">{relativeTime(page.wpModifiedAt)}</span>
                                </TableCell>
                                <TableCell>
                                    <Button
                                        size="sm"
                                        variant="ghost"
                                        icon={OpenInNewIcon}
                                        disabled={!isBrowsable(url)}
                                        title={copy.app.openExternal}
                                        onClick={() => {
                                            void openExternal(url);
                                        }}
                                    >
                                        {copy.app.openExternal}
                                    </Button>
                                </TableCell>
                            </TableRow>
                        );
                    })}
                </DenseTable>
            )}
        </Panel>
    );
}
