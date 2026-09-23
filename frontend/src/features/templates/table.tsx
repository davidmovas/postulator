import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { Template } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    Button,
    DenseTable,
    DashboardCustomizeIcon,
    EmptyState,
    FilterAltIcon,
    SkeletonRows,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
    VerifiedIcon,
} from "../../ui/index.js";
import { pageKindLabel, scopeLabel, scopeTone } from "./labels.js";

const columns = "minmax(104px,1fr) 88px 96px 52px 64px 96px";

interface RowProps {
    template: Template;
    selected: boolean;
    isDefault: boolean;
    onSelect: (id: string) => void;
    onOpen: (id: string) => void;
}

function Row({ template, selected, isDefault, onSelect, onOpen }: RowProps): ReactElement {
    return (
        <TableRow
            interactive={true}
            selected={selected}
            tabIndex={0}
            data-template-row={true}
            data-template-id={template.id}
            onClick={() => {
                onSelect(template.id);
            }}
            onDoubleClick={() => {
                onOpen(template.id);
            }}
            onKeyDown={(event) => {
                if (event.key === "Enter") {
                    event.preventDefault();
                    onOpen(template.id);
                }
                if (event.key === " ") {
                    event.preventDefault();
                    onSelect(template.id);
                }
            }}
        >
            <TableCell>
                <span className="truncate font-medium text-ink">{template.name}</span>
            </TableCell>
            <TableCell>
                <StatusBadge tone={scopeTone(template.scope)} dot={false}>
                    {scopeLabel(template.scope)}
                </StatusBadge>
            </TableCell>
            <TableCell muted={true}>{pageKindLabel(template.pageKind)}</TableCell>
            <TableCell mono={true} muted={true}>
                {copy.templates.versionLabel(template.version)}
            </TableCell>
            <TableCell>
                {isDefault ? (
                    <span className="text-accent" title={copy.templates.siteDefault}>
                        <VerifiedIcon size={15} aria-label={copy.templates.siteDefault} role="img" />
                    </span>
                ) : null}
            </TableCell>
            <TableCell mono={true} muted={true} align="right" title={absoluteTime(template.updatedAt)}>
                {relativeTime(template.updatedAt)}
            </TableCell>
        </TableRow>
    );
}

export interface TemplateTableProps {
    rows: readonly Template[];
    pending: boolean;
    narrowed: boolean;
    selectedId: string | null;
    defaultId: string | null;
    hasMore: boolean;
    loadingMore: boolean;
    onSelect: (id: string) => void;
    onOpen: (id: string) => void;
    onLoadMore: () => void;
    onCreate: () => void;
    onAskAgent: () => void;
    onClearFilters: () => void;
}

export function TemplateTable({
    rows,
    pending,
    narrowed,
    selectedId,
    defaultId,
    hasMore,
    loadingMore,
    onSelect,
    onOpen,
    onLoadMore,
    onCreate,
    onAskAgent,
    onClearFilters,
}: TemplateTableProps): ReactElement {
    if (pending) {
        return (
            <div className="p-4">
                <SkeletonRows rows={10} label={copy.templates.loading} />
            </div>
        );
    }

    if (rows.length === 0) {
        return (
            <div className="flex items-start justify-center p-6">
                {narrowed ? (
                    <EmptyState
                        icon={FilterAltIcon}
                        title={copy.templates.noMatch}
                        body={copy.templates.noMatchBody}
                        actions={<Button onClick={onClearFilters}>{copy.templates.clearFilters}</Button>}
                    />
                ) : (
                    <EmptyState
                        icon={DashboardCustomizeIcon}
                        title={copy.empty.templates}
                        actions={
                            <>
                                <Button variant="primary" onClick={onCreate}>
                                    {copy.templates.newTemplate}
                                </Button>
                                <Button onClick={onAskAgent}>{copy.templates.askAgent}</Button>
                            </>
                        }
                    />
                )}
            </div>
        );
    }

    return (
        <div className="min-h-0 flex-1 overflow-auto">
            <DenseTable columns={columns} label={copy.templates.title}>
                <TableHead>
                    <div>{copy.templates.columns.name}</div>
                    <div>{copy.templates.columns.scope}</div>
                    <div>{copy.templates.columns.pageKind}</div>
                    <div>{copy.templates.columns.version}</div>
                    <div>{copy.templates.columns.default}</div>
                    <div className="text-right">{copy.templates.columns.updated}</div>
                </TableHead>
                {rows.map((template) => (
                    <Row
                        key={template.id}
                        template={template}
                        selected={template.id === selectedId}
                        isDefault={template.id === defaultId}
                        onSelect={onSelect}
                        onOpen={onOpen}
                    />
                ))}
                {hasMore ? (
                    <TableRow className="py-1" style={{ height: "auto" }}>
                        <div role="cell" className="col-span-full flex justify-center">
                            <Button size="sm" busy={loadingMore} onClick={onLoadMore}>
                                {copy.app.loadMore}
                            </Button>
                        </div>
                    </TableRow>
                ) : null}
            </DenseTable>
        </div>
    );
}
