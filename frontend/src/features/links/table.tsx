import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { PageAudit } from "../../data/types.js";
import {
    Button,
    cx,
    DenseTable,
    EmptyState,
    HubIcon,
    LinkOffIcon,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
    toneClasses,
    VirtualRows,
} from "../../ui/index.js";
import { pageStatusLabel, statusTone } from "../pages/labels.js";
import { severityTone } from "./labels.js";
import { severityOf } from "./model/audit.js";

const columns = "minmax(160px, 2.2fr) minmax(120px, 1.4fr) 64px 96px 64px 64px 76px 84px";
const rowHeight = 28;

export interface AuditTableProps {
    siteId: string;
    rows: readonly PageAudit[];
    selectedId: string | null;
    narrowed: boolean;
    onOpen: (pageId: string) => void;
    onReset: () => void;
    onOpenGraph: () => void;
}

export function AuditTable({ siteId, rows, selectedId, narrowed, onOpen, onReset, onOpenGraph }: AuditTableProps): ReactElement {
    if (rows.length === 0) {
        return (
            <div className="flex flex-1 items-start justify-center p-6">
                {narrowed ? (
                    <EmptyState
                        icon={LinkOffIcon}
                        title={copy.links.empty.noMatch}
                        body={copy.links.empty.noMatchBody}
                        actions={
                            <Button size="sm" variant="secondary" onClick={onReset}>
                                {copy.links.filters.reset}
                            </Button>
                        }
                    />
                ) : (
                    <EmptyState
                        icon={HubIcon}
                        title={copy.links.empty.title}
                        body={copy.links.empty.body}
                        actions={
                            <Button size="sm" variant="secondary" onClick={onOpenGraph}>
                                {copy.links.empty.graph}
                            </Button>
                        }
                    />
                )}
            </div>
        );
    }

    const row = (position: number): ReactElement => {
        const held = rows[position];
        const severity = severityOf(held);
        const skipped = held.skipReason !== "";
        return (
            <TableRow
                key={held.pageId}
                interactive={true}
                selected={held.pageId === selectedId}
                tabIndex={0}
                onClick={() => {
                    onOpen(held.pageId);
                }}
                onKeyDown={(event) => {
                    if (event.key === "Enter") {
                        event.preventDefault();
                        onOpen(held.pageId);
                    }
                }}
            >
                <TableCell mono={true}>
                    <span className="flex min-w-0 items-center gap-2">
                        <span className={cx("h-1.5 w-1.5 shrink-0 rounded-full", toneClasses[severityTone(severity)].solid)} aria-hidden={true} />
                        <span className="truncate">{held.path}</span>
                    </span>
                </TableCell>
                <TableCell muted={held.entityName === ""}>
                    {held.entityName === "" ? (copy.links.cell.skipped[held.skipReason] ?? copy.links.cell.skipped.unmapped) : held.entityName}
                </TableCell>
                <TableCell mono={true} align="right" muted={skipped}>
                    {skipped ? "" : copy.links.cell.ofTargets(held.satisfied, held.targets)}
                </TableCell>
                <TableCell>
                    {held.missing === 0 ? null : (
                        <span className="flex items-center gap-1">
                            <span className={cx("font-mono text-xs", held.missingRequired > 0 ? "text-danger" : "text-warn")}>{held.missing}</span>
                            {held.missingRequired > 0 ? (
                                <StatusBadge tone="danger" dot={false}>
                                    {copy.links.cell.required(held.missingRequired)}
                                </StatusBadge>
                            ) : null}
                        </span>
                    )}
                </TableCell>
                <TableCell mono={true} align="right" className={cx(held.blocked > 0 && "text-danger")}>
                    {held.blocked === 0 ? "" : String(held.blocked)}
                </TableCell>
                <TableCell mono={true} align="right" className={cx(held.offGraph > 0 && "text-warn")}>
                    {held.offGraph === 0 ? "" : String(held.offGraph)}
                </TableCell>
                <TableCell>
                    {held.orphan ? (
                        <StatusBadge tone="warn" dot={false}>
                            {copy.links.cell.orphan}
                        </StatusBadge>
                    ) : (
                        <span className="font-mono text-xs text-ink-dim">{held.inbound}</span>
                    )}
                </TableCell>
                <TableCell>
                    <StatusBadge tone={statusTone(held.status)}>{pageStatusLabel(held.status)}</StatusBadge>
                </TableCell>
            </TableRow>
        );
    };

    return (
        <DenseTable columns={columns} label={copy.links.title} className="flex min-h-0 flex-1 flex-col">
            <TableHead>
                <TableCell>{copy.links.columns.path}</TableCell>
                <TableCell>{copy.links.columns.entity}</TableCell>
                <TableCell align="right">{copy.links.columns.links}</TableCell>
                <TableCell>{copy.links.columns.missing}</TableCell>
                <TableCell align="right">{copy.links.columns.blocked}</TableCell>
                <TableCell align="right">{copy.links.columns.offGraph}</TableCell>
                <TableCell>{copy.links.columns.inbound}</TableCell>
                <TableCell>{copy.links.columns.status}</TableCell>
            </TableHead>
            <VirtualRows count={rows.length} rowHeight={rowHeight} row={row} scrollKey={`${siteId}:links`} />
        </DenseTable>
    );
}
