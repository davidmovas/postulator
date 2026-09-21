import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { pageStatuses } from "../../generated/vocab.js";
import { Button, cx, SectionLabel, Segmented, Select } from "../../ui/index.js";
import type { SegmentedOption, SelectOption } from "../../ui/index.js";
import type { GraphIndex } from "../graph/model/index.js";
import { showLabel } from "./labels.js";
import type { LinksQuery, LinksSort, Show } from "./model/params.js";
import { defaultQuery, narrowed, shows } from "./model/params.js";

const anyEntity = "any";

const sortOptions: readonly SegmentedOption<LinksSort>[] = [
    { value: "severity", label: copy.links.filters.severity },
    { value: "path", label: copy.links.filters.path },
];

interface CountedRowProps {
    label: string;
    count: number | undefined;
    active: boolean;
    onSelect: () => void;
}

function CountedRow({ label, count, active, onSelect }: CountedRowProps): ReactElement {
    return (
        <button
            type="button"
            aria-pressed={active}
            onClick={onSelect}
            className={cx(
                "flex h-6 w-full items-center justify-between gap-2 rounded-sm px-1.5 text-xs transition-colors duration-100",
                active ? "bg-accent-soft text-ink" : "text-ink-dim hover:bg-inset hover:text-ink",
            )}
        >
            <span className="truncate">{label}</span>
            <span className="shrink-0 font-mono text-2xs text-ink-faint">{count ?? ""}</span>
        </button>
    );
}

export interface LinkFiltersProps {
    query: LinksQuery;
    counts: Record<Show, number> | null;
    statusCounts: ReadonlyMap<string, number>;
    index: GraphIndex | null;
    onChange: (next: LinksQuery) => void;
}

export function LinkFilters({ query, counts, statusCounts, index, onChange }: LinkFiltersProps): ReactElement {
    const entityOptions: SelectOption<string>[] = [
        { value: anyEntity, label: copy.links.filters.anyEntity },
        ...(index?.entities ?? []).map((entity) => ({ value: entity.id, label: entity.name })),
    ];

    return (
        <div aria-label={copy.links.filters.title} className="flex flex-col gap-3 p-3">
            <div className="flex items-center justify-between gap-2">
                <SectionLabel>{copy.links.filters.title}</SectionLabel>
                <Button
                    size="sm"
                    variant="ghost"
                    disabled={!narrowed(query)}
                    onClick={() => {
                        onChange({ ...defaultQuery, sort: query.sort });
                    }}
                >
                    {copy.links.filters.reset}
                </Button>
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.links.filters.show}</SectionLabel>
                {shows.map((show) => (
                    <CountedRow
                        key={show}
                        label={showLabel(show)}
                        count={counts?.[show]}
                        active={query.show === show}
                        onSelect={() => {
                            onChange({ ...query, show });
                        }}
                    />
                ))}
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.links.filters.status}</SectionLabel>
                <CountedRow
                    label={copy.links.filters.anyStatus}
                    count={counts?.all}
                    active={query.status === ""}
                    onSelect={() => {
                        onChange({ ...query, status: "" });
                    }}
                />
                {pageStatuses.map((status) => (
                    <CountedRow
                        key={status}
                        label={status}
                        count={statusCounts.get(status)}
                        active={query.status === status}
                        onSelect={() => {
                            onChange({ ...query, status: query.status === status ? "" : status });
                        }}
                    />
                ))}
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.links.filters.entity}</SectionLabel>
                <Select
                    value={query.entity === "" ? anyEntity : query.entity}
                    options={entityOptions}
                    aria-label={copy.links.filters.entity}
                    disabled={index === null || index.entities.length === 0}
                    onValueChange={(next) => {
                        onChange({ ...query, entity: next === anyEntity ? "" : next });
                    }}
                />
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.links.filters.sort}</SectionLabel>
                <Segmented
                    label={copy.links.filters.sort}
                    value={query.sort}
                    options={sortOptions}
                    onValueChange={(sort) => {
                        onChange({ ...query, sort });
                    }}
                />
            </div>
        </div>
    );
}
