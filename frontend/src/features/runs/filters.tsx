import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { runKinds, runStatuses } from "../../generated/vocab.js";
import { Button, cx, SectionLabel } from "../../ui/index.js";
import { statusLabel } from "./labels.js";
import { defaultQuery, narrowed } from "./params.js";
import type { RunsQuery } from "./params.js";

interface FilterRowProps {
    label: string;
    active: boolean;
    onSelect: () => void;
}

function FilterRow({ label, active, onSelect }: FilterRowProps): ReactElement {
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
        </button>
    );
}

export interface RunFiltersProps {
    query: RunsQuery;
    onChange: (next: RunsQuery) => void;
}

export function RunFilters({ query, onChange }: RunFiltersProps): ReactElement {
    return (
        <aside
            aria-label={copy.runs.filters.title}
            className="flex w-48 shrink-0 flex-col gap-3 overflow-auto border-r border-hairline bg-panel p-3"
        >
            <div className="flex items-center justify-between gap-2">
                <SectionLabel>{copy.runs.filters.title}</SectionLabel>
                <Button
                    size="sm"
                    variant="ghost"
                    disabled={!narrowed(query)}
                    onClick={() => {
                        onChange({ ...defaultQuery, sort: query.sort });
                    }}
                >
                    {copy.runs.filters.reset}
                </Button>
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.runs.filters.status}</SectionLabel>
                <FilterRow
                    label={copy.runs.filters.anyStatus}
                    active={query.status === ""}
                    onSelect={() => {
                        onChange({ ...query, status: "" });
                    }}
                />
                {runStatuses.map((status) => (
                    <FilterRow
                        key={status}
                        label={statusLabel(status)}
                        active={query.status === status}
                        onSelect={() => {
                            onChange({ ...query, status: query.status === status ? "" : status });
                        }}
                    />
                ))}
            </div>

            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.runs.filters.kind}</SectionLabel>
                <FilterRow
                    label={copy.runs.filters.anyKind}
                    active={query.kind === ""}
                    onSelect={() => {
                        onChange({ ...query, kind: "" });
                    }}
                />
                {runKinds.map((kind) => (
                    <FilterRow
                        key={kind}
                        label={kind}
                        active={query.kind === kind}
                        onSelect={() => {
                            onChange({ ...query, kind: query.kind === kind ? "" : kind });
                        }}
                    />
                ))}
            </div>
        </aside>
    );
}
