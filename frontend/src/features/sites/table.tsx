import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { isBrowsable, openExternal } from "../../data/host.js";
import type { SiteSort } from "../../data/sorts.js";
import type { Site } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    DenseTable,
    ExtensionIcon,
    ExtensionOffIcon,
    IconButton,
    OpenInNewIcon,
    SkeletonRows,
    SortableHeader,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
} from "../../ui/index.js";
import { siteStatusTone } from "./status.js";

const columns = "minmax(8rem,1.4fr) minmax(10rem,2fr) 5.5rem 7rem 6rem";

export function nextSort(current: SiteSort | null, field: SiteSort["field"]): SiteSort {
    if (current !== null && current.field === field) {
        return { field, desc: !current.desc };
    }
    return { field, desc: field === "createdAt" };
}

export interface SiteTableProps {
    rows: readonly Site[];
    loading: boolean;
    selectedId: string | null;
    sort: SiteSort | null;
    onSortChange: (sort: SiteSort) => void;
    onSelect: (id: string) => void;
    onEnter: (id: string) => void;
}

export function SiteTable({
    rows,
    loading,
    selectedId,
    sort,
    onSortChange,
    onSelect,
    onEnter,
}: SiteTableProps): ReactElement {
    return (
        <DenseTable columns={columns} label={copy.sites.title}>
            <TableHead>
                <SortableHeader
                    active={sort?.field === "name"}
                    direction={sort?.desc === true ? "desc" : "asc"}
                    onToggle={() => {
                        onSortChange(nextSort(sort, "name"));
                    }}
                >
                    {copy.sites.columns.name}
                </SortableHeader>
                <span role="columnheader">{copy.sites.columns.baseUrl}</span>
                <span role="columnheader">{copy.sites.columns.status}</span>
                <span role="columnheader">{copy.sites.columns.plugin}</span>
                <SortableHeader
                    active={sort?.field === "createdAt"}
                    direction={sort?.desc === true ? "desc" : "asc"}
                    align="right"
                    onToggle={() => {
                        onSortChange(nextSort(sort, "createdAt"));
                    }}
                >
                    {copy.sites.columns.created}
                </SortableHeader>
            </TableHead>
            {loading ? (
                <SkeletonRows rows={6} label={copy.app.loading} className="p-3" />
            ) : (
                rows.map((row) => (
                    <TableRow
                        key={row.id}
                        data-site-id={row.id}
                        interactive={true}
                        selected={row.id === selectedId}
                        tabIndex={0}
                        onClick={() => {
                            onSelect(row.id);
                        }}
                        onKeyDown={(event) => {
                            if (event.key === "Enter") {
                                event.preventDefault();
                                onEnter(row.id);
                                return;
                            }
                            if (event.key === " ") {
                                event.preventDefault();
                                onSelect(row.id);
                            }
                        }}
                    >
                        <TableCell>
                            <Link
                                to={`/s/${row.id}/overview`}
                                title={copy.sites.open}
                                onClick={(event) => {
                                    event.stopPropagation();
                                }}
                                className="truncate text-ink hover:text-accent hover:underline"
                            >
                                {row.name}
                            </Link>
                        </TableCell>
                        <TableCell mono={true} muted={true} title={row.baseUrl}>
                            <span className="flex min-w-0 items-center gap-1">
                                <span className="truncate">{row.baseUrl}</span>
                                {isBrowsable(row.baseUrl) ? (
                                    <IconButton
                                        icon={OpenInNewIcon}
                                        label={copy.app.openExternal}
                                        variant="ghost"
                                        size="sm"
                                        onClick={(event) => {
                                            event.stopPropagation();
                                            void openExternal(row.baseUrl);
                                        }}
                                    />
                                ) : null}
                            </span>
                        </TableCell>
                        <TableCell>
                            <StatusBadge tone={siteStatusTone(row.status)}>{row.status}</StatusBadge>
                        </TableCell>
                        <TableCell>
                            {row.plugin.installed ? (
                                <StatusBadge tone="ok" icon={ExtensionIcon} dot={false}>
                                    {row.plugin.version === ""
                                        ? copy.shell.pluginInstalled
                                        : row.plugin.version}
                                </StatusBadge>
                            ) : (
                                <StatusBadge tone="warn" icon={ExtensionOffIcon} dot={false}>
                                    {copy.shell.pluginMissing}
                                </StatusBadge>
                            )}
                        </TableCell>
                        <TableCell align="right" muted={true} title={absoluteTime(row.createdAt)}>
                            {relativeTime(row.createdAt)}
                        </TableCell>
                    </TableRow>
                ))
            )}
        </DenseTable>
    );
}
