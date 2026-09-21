import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    ChevronRightIcon,
    cx,
    DescriptionIcon,
    LinkOffIcon,
    StatusBadge,
    SyncProblemIcon,
    TableCell,
    TableRow,
} from "../../ui/index.js";
import { pageStatusLabel, statusTone } from "./labels.js";
import type { TreeRow } from "./tree-model.js";

export const treeColumns = "minmax(200px,3fr) 92px minmax(96px,1.5fr) 44px 84px";
export const indentStep = 14;

export interface PageTreeRowProps {
    row: Extract<TreeRow, { kind: "page" }>;
    selected: boolean;
    entityName: string | null;
    onToggle: (pageId: string) => void;
    onSelect: (pageId: string) => void;
    onOpen: (pageId: string) => void;
}

export function PageTreeRow({
    row,
    selected,
    entityName,
    onToggle,
    onSelect,
    onOpen,
}: PageTreeRowProps): ReactElement {
    const { page } = row;
    return (
        <TableRow
            data-page-row={true}
            data-page-id={page.id}
            data-page-status={page.status}
            interactive={true}
            selected={selected}
            tabIndex={0}
            onClick={() => {
                onSelect(page.id);
            }}
            onDoubleClick={() => {
                onOpen(page.id);
            }}
            onKeyDown={(event) => {
                if (event.key === "Enter") {
                    event.preventDefault();
                    onOpen(page.id);
                }
            }}
        >
            <TableCell>
                <span
                    className="flex min-w-0 items-center gap-1"
                    style={{ paddingLeft: `${row.depth * indentStep}px` }}
                >
                    {row.childCount === 0 ? (
                        <DescriptionIcon size={13} className="shrink-0 text-ink-faint" />
                    ) : (
                        <button
                            type="button"
                            aria-expanded={row.expanded}
                            aria-label={page.path}
                            className="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-raised hover:text-ink"
                            onClick={(event) => {
                                event.stopPropagation();
                                onToggle(page.id);
                            }}
                        >
                            <ChevronRightIcon
                                size={14}
                                className={cx("transition-transform duration-100", row.expanded && "rotate-90")}
                            />
                        </button>
                    )}
                    <span className="truncate font-mono text-xs text-ink">{page.path}</span>
                    <span className="truncate text-ink-dim">
                        {page.title === "" ? copy.pages.untitled : page.title}
                    </span>
                    {row.childCount === 0 ? null : (
                        <span className="shrink-0 font-mono text-2xs text-ink-faint">
                            {copy.pages.tree.children(row.childCount)}
                        </span>
                    )}
                </span>
            </TableCell>
            <TableCell>
                <StatusBadge tone={statusTone(page.status)}>{pageStatusLabel(page.status)}</StatusBadge>
            </TableCell>
            <TableCell muted={page.entityId === null}>
                {page.entityId === null ? (
                    <span className="flex min-w-0 items-center gap-1 text-ink-faint">
                        <LinkOffIcon size={13} className="shrink-0" />
                        <span className="truncate">{copy.pages.unmapped}</span>
                    </span>
                ) : (
                    (entityName ?? copy.pages.mapped)
                )}
            </TableCell>
            <TableCell>
                {page.drift ? (
                    <span className="flex items-center gap-1 text-warn" title={copy.pages.drift.title}>
                        <SyncProblemIcon size={14} className="shrink-0" />
                        <span className="sr-only">{copy.pages.drift.badge}</span>
                    </span>
                ) : null}
            </TableCell>
            <TableCell mono={true} muted={true} title={absoluteTime(page.lastSyncedAt)}>
                {relativeTime(page.lastSyncedAt)}
            </TableCell>
        </TableRow>
    );
}
