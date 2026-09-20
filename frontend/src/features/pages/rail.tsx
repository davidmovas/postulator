import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { useSiteOverview } from "../../data/hooks/reports.js";
import { pageStatuses } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { Button, Checkbox, cx, SectionLabel, Select } from "../../ui/index.js";
import type { EntityIndex } from "./entities.js";
import { defaultQuery, narrowed } from "./params.js";
import type { PagesQuery } from "./params.js";

const anyEntity = "any";

interface CountedRowProps {
    label: string;
    count: number | undefined;
    active: boolean;
    disabled: boolean;
    onSelect: () => void;
}

function CountedRow({ label, count, active, disabled, onSelect }: CountedRowProps): ReactElement {
    return (
        <button
            type="button"
            aria-pressed={active}
            disabled={disabled}
            onClick={onSelect}
            className={cx(
                "flex h-6 w-full items-center justify-between gap-2 rounded-sm px-1.5 text-xs transition-colors duration-100",
                "disabled:cursor-not-allowed disabled:opacity-50",
                active ? "bg-accent-soft text-ink" : "text-ink-dim enabled:hover:bg-inset enabled:hover:text-ink",
            )}
        >
            <span className="truncate">{label}</span>
            <span className="shrink-0 font-mono text-2xs text-ink-faint">{count ?? ""}</span>
        </button>
    );
}

export interface PageRailProps {
    siteId: string;
    query: PagesQuery;
    index: EntityIndex;
    disabled: boolean;
    onChange: (next: PagesQuery) => void;
}

export function PageRail({ siteId, query, index, disabled, onChange }: PageRailProps): ReactElement {
    const overview = useSiteOverview(siteId);
    const totals = overview.data?.pages;

    const entityOptions: SelectOption<string>[] = [
        { value: anyEntity, label: copy.pages.filters.anyEntity },
        ...index.entities.map((entity) => ({ value: entity.id, label: entity.name })),
    ];

    return (
        <div
            aria-label={copy.pages.filters.title}
            title={disabled ? copy.pages.tableOnly : undefined}
            className="flex flex-col gap-3 p-3"
        >
            <div className="flex items-center justify-between gap-2">
                <SectionLabel>{copy.pages.filters.title}</SectionLabel>
                <Button
                    size="sm"
                    variant="ghost"
                    disabled={disabled || !narrowed(query)}
                    onClick={() => {
                        onChange({ ...defaultQuery, view: query.view, sort: query.sort });
                    }}
                >
                    {copy.pages.filters.reset}
                </Button>
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.pages.filters.status}</SectionLabel>
                <CountedRow
                    label={copy.pages.filters.anyStatus}
                    count={totals?.total}
                    active={query.status === ""}
                    disabled={disabled}
                    onSelect={() => {
                        onChange({ ...query, status: "" });
                    }}
                />
                {pageStatuses.map((status) => (
                    <CountedRow
                        key={status}
                        label={status}
                        count={totals?.byStatus?.[status]}
                        active={query.status === status}
                        disabled={disabled}
                        onSelect={() => {
                            onChange({ ...query, status: query.status === status ? "" : status });
                        }}
                    />
                ))}
            </div>

            <div className="flex flex-col gap-1.5">
                <SectionLabel>{copy.pages.filters.entity}</SectionLabel>
                <div title={index.entities.length === 0 ? copy.pages.filters.noEntities : undefined}>
                    <Select
                        value={query.entityId === "" ? anyEntity : query.entityId}
                        options={entityOptions}
                        aria-label={copy.pages.filters.entity}
                        disabled={disabled || index.entities.length === 0}
                        onValueChange={(next) => {
                            onChange({ ...query, entityId: next === anyEntity ? "" : next, unmapped: false });
                        }}
                    />
                </div>
                <div className="flex items-center justify-between gap-2">
                    <Checkbox
                        checked={query.unmapped}
                        disabled={disabled}
                        label={copy.pages.filters.unmapped}
                        onChange={(event) => {
                            onChange({
                                ...query,
                                unmapped: event.target.checked,
                                entityId: event.target.checked ? "" : query.entityId,
                            });
                        }}
                    />
                    <span className="shrink-0 font-mono text-2xs text-ink-faint">{totals?.unmapped ?? ""}</span>
                </div>
            </div>
        </div>
    );
}
