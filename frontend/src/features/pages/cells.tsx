import type { ReactElement, ReactNode } from "react";

import { copy } from "../../copy/index.js";
import type { Page } from "../../data/types.js";
import { pageCategoryItems } from "../../domain/categories.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { CategoryTrail, LinkOffIcon, SyncProblemIcon, TableCell, TableRow } from "../../ui/index.js";

export interface PageTableRowProps {
    page: Pick<Page, "id" | "status">;
    selected: boolean;
    onSelect: (pageId: string) => void;
    onOpen: (pageId: string) => void;
    children: ReactNode;
}

export function PageTableRow({ page, selected, onSelect, onOpen, children }: PageTableRowProps): ReactElement {
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
            {children}
        </TableRow>
    );
}

export interface EntityCellProps {
    entityId: string | null;
    entityName: string | null;
}

export function EntityCell({ entityId, entityName }: EntityCellProps): ReactElement {
    return (
        <TableCell muted={entityId === null}>
            {entityId === null ? (
                <span className="flex min-w-0 items-center gap-1 text-ink-faint">
                    <LinkOffIcon size={13} className="shrink-0" />
                    <span className="truncate">{copy.pages.unmapped}</span>
                </span>
            ) : (
                (entityName ?? copy.pages.mapped)
            )}
        </TableCell>
    );
}

export interface CategoryCellProps {
    page: Pick<Page, "categories" | "categoriesNeedPlugin">;
}

export function CategoryCell({ page }: CategoryCellProps): ReactElement {
    return (
        <TableCell>
            <CategoryTrail items={pageCategoryItems(page)} label={copy.categories.trail} compact={true} />
        </TableCell>
    );
}

export interface DriftCellProps {
    drift: boolean;
}

export function DriftCell({ drift }: DriftCellProps): ReactElement {
    return (
        <TableCell>
            {drift ? (
                <span className="flex items-center gap-1 text-warn" title={copy.pages.drift.title}>
                    <SyncProblemIcon size={14} className="shrink-0" />
                    <span className="sr-only">{copy.pages.drift.badge}</span>
                </span>
            ) : null}
        </TableCell>
    );
}

export interface SyncedCellProps {
    at: string | null;
}

export function SyncedCell({ at }: SyncedCellProps): ReactElement {
    return (
        <TableCell mono={true} muted={true} title={absoluteTime(at)}>
            {relativeTime(at)}
        </TableCell>
    );
}
