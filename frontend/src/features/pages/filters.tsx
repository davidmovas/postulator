import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { useSiteOverview } from "../../data/hooks/reports.js";
import { copy } from "../../copy/index.js";
import { pageStatuses } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { Button, Checkbox, cx, Input, SectionLabel, Select } from "../../ui/index.js";
import { defaultQuery, narrowed } from "./params.js";
import type { PagesQuery } from "./params.js";
import type { EntityIndex } from "./entities.js";

const anyEntity = "any";

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

export interface PageFiltersProps {
    siteId: string;
    query: PagesQuery;
    onChange: (next: PagesQuery) => void;
    index: EntityIndex;
}

export function PageFilters({ siteId, query, onChange, index }: PageFiltersProps): ReactElement {
    const overview = useSiteOverview(siteId);
    const totals = overview.data?.pages;
    const [prefix, setPrefix] = useState(query.pathPrefix);

    useEffect(() => {
        setPrefix(query.pathPrefix);
    }, [query.pathPrefix]);

    const applyPrefix = (): void => {
        const trimmed = prefix.trim();
        if (trimmed !== query.pathPrefix) {
            onChange({ ...query, pathPrefix: trimmed });
        }
    };

    const entityOptions: SelectOption<string>[] = [
        { value: anyEntity, label: copy.pages.filters.anyEntity },
        ...index.entities.map((entity) => ({ value: entity.id, label: entity.name })),
    ];

    return (
        <aside
            aria-label={copy.pages.filters.title}
            className="flex w-52 shrink-0 flex-col gap-3 overflow-auto border-r border-hairline bg-panel p-3"
        >
            <div className="flex items-center justify-between gap-2">
                <SectionLabel>{copy.pages.filters.title}</SectionLabel>
                <Button
                    size="sm"
                    variant="ghost"
                    disabled={!narrowed(query)}
                    onClick={() => {
                        onChange({ ...defaultQuery, view: query.view, sort: query.sort });
                    }}
                >
                    {copy.pages.filters.reset}
                </Button>
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.pages.filters.pathPrefix}</SectionLabel>
                <Input
                    mono={true}
                    value={prefix}
                    placeholder="/"
                    aria-label={copy.pages.filters.pathPrefix}
                    onChange={(event) => {
                        setPrefix(event.target.value);
                    }}
                    onBlur={applyPrefix}
                    onKeyDown={(event) => {
                        if (event.key === "Enter") {
                            applyPrefix();
                        }
                    }}
                />
                <p className="text-2xs text-ink-faint">{copy.pages.filters.pathPrefixHint}</p>
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.pages.filters.status}</SectionLabel>
                <CountedRow
                    label={copy.pages.filters.anyStatus}
                    count={totals?.total}
                    active={query.status === ""}
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
                        onSelect={() => {
                            onChange({ ...query, status: query.status === status ? "" : status });
                        }}
                    />
                ))}
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.pages.filters.entity}</SectionLabel>
                <Select
                    value={query.entityId === "" ? anyEntity : query.entityId}
                    options={entityOptions}
                    aria-label={copy.pages.filters.entity}
                    disabled={index.entities.length === 0}
                    onValueChange={(next) => {
                        onChange({ ...query, entityId: next === anyEntity ? "" : next, unmapped: false });
                    }}
                />
                {index.entities.length === 0 ? (
                    <p className="text-2xs text-ink-faint">{copy.pages.filters.noEntities}</p>
                ) : null}
                {index.complete ? null : (
                    <p className="text-2xs text-warn">{copy.pages.filters.partialEntities}</p>
                )}
                <div className="mt-1 flex items-center justify-between gap-2">
                    <Checkbox
                        checked={query.unmapped}
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
        </aside>
    );
}
