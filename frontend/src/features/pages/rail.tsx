import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { usePagePaths } from "../../data/hooks/pages.js";
import { useSiteOverview } from "../../data/hooks/reports.js";
import { pageStatuses } from "../../generated/vocab.js";
import type { SegmentedOption, SelectOption } from "../../ui/index.js";
import { Button, Checkbox, cx, Input, SectionLabel, Segmented, Select } from "../../ui/index.js";
import { PageSearch } from "../findings/page-search.js";
import type { EntityIndex } from "./entities.js";
import { pageStatusLabel } from "./labels.js";
import { defaultQuery, narrowed } from "./params.js";
import type { BodyChoice, PagesQuery } from "./params.js";
import type { TypeNaming } from "./views/naming.js";

const anyEntity = "any";

const anyType = "any";

const said = copy.pages.listFilters;

const bodyOptions: readonly SegmentedOption<BodyChoice | "any">[] = [
    { value: "any", label: said.bodies.any },
    { value: "with", label: said.bodies.with },
    { value: "without", label: said.bodies.without },
];

function PrefixFilter({ query, onChange }: { query: PagesQuery; onChange: (next: PagesQuery) => void }): ReactElement {
    const [prefix, setPrefix] = useState(query.pathPrefix);

    useEffect(() => {
        setPrefix(query.pathPrefix);
    }, [query.pathPrefix]);

    const apply = (): void => {
        const trimmed = prefix.trim();
        if (trimmed !== query.pathPrefix) {
            onChange({ ...query, pathPrefix: trimmed });
        }
    };

    return (
        <Input
            type="search"
            mono={true}
            size="sm"
            value={prefix}
            placeholder="/"
            aria-label={said.prefix}
            onChange={(event) => {
                setPrefix(event.target.value);
            }}
            onBlur={apply}
            onKeyDown={(event) => {
                if (event.key === "Enter") {
                    apply();
                }
            }}
        />
    );
}

function BranchFilter({ siteId, query, onChange }: { siteId: string; query: PagesQuery; onChange: (next: PagesQuery) => void }): ReactElement {
    const [picking, setPicking] = useState(false);
    const ids = query.branch === "" ? [] : [query.branch];
    const paths = usePagePaths(ids, ids.length > 0);
    if (query.branch !== "") {
        return (
            <div className="flex min-w-0 items-center gap-1.5">
                <span data-pages-branch={query.branch} className="min-w-0 flex-1 truncate font-mono text-xs text-ink-soft">
                    {paths.get(query.branch) ?? ""}
                </span>
                <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => {
                        onChange({ ...query, branch: "" });
                    }}
                >
                    {said.branchClear}
                </Button>
            </div>
        );
    }
    return picking ? (
        <PageSearch
            siteId={siteId}
            picked={null}
            onPick={(page) => {
                setPicking(false);
                onChange({ ...query, branch: page.pageId });
            }}
        />
    ) : (
        <div className="flex min-w-0 items-center gap-1.5">
            <span className="min-w-0 flex-1 truncate text-xs text-ink-dim">{said.branchAnywhere}</span>
            <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                    setPicking(true);
                }}
            >
                {said.branchPick}
            </Button>
        </div>
    );
}

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

export interface PageRailProps {
    siteId: string;
    query: PagesQuery;
    index: EntityIndex;
    naming: TypeNaming;
    onChange: (next: PagesQuery) => void;
}

export function PageRail({ siteId, query, index, naming, onChange }: PageRailProps): ReactElement {
    const overview = useSiteOverview(siteId);
    const totals = overview.data?.pages;

    const typeOptions: SelectOption<string>[] = [
        { value: anyType, label: said.anyType },
        ...naming.types.map((type) => ({ value: type.id, label: type.name })),
    ];

    const entityOptions: SelectOption<string>[] = [
        { value: anyEntity, label: copy.pages.filters.anyEntity },
        ...index.entities.map((entity) => ({ value: entity.id, label: index.labels.get(entity.id) ?? entity.name })),
    ];

    return (
        <div aria-label={copy.pages.filters.title} className="flex flex-col gap-3 p-3">
            <div className="flex items-center justify-between gap-2">
                <SectionLabel>{copy.pages.filters.title}</SectionLabel>
                <Button
                    size="sm"
                    variant="ghost"
                    disabled={!narrowed(query)}
                    onClick={() => {
                        onChange({ ...defaultQuery, sort: query.sort });
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
                    onSelect={() => {
                        onChange({ ...query, status: "" });
                    }}
                />
                {pageStatuses.map((status) => (
                    <CountedRow
                        key={status}
                        label={pageStatusLabel(status)}
                        count={totals?.byStatus?.[status]}
                        active={query.status === status}
                        onSelect={() => {
                            onChange({ ...query, status: query.status === status ? "" : status });
                        }}
                    />
                ))}
            </div>

            <div className="flex flex-col gap-1.5">
                <SectionLabel>{said.type}</SectionLabel>
                <Select
                    value={query.typeId === "" ? anyType : query.typeId}
                    options={typeOptions}
                    aria-label={said.type}
                    onValueChange={(next) => {
                        onChange({ ...query, typeId: next === anyType ? "" : next });
                    }}
                />
            </div>

            <div className="flex flex-col gap-1.5">
                <SectionLabel>{said.body}</SectionLabel>
                <div>
                    <Segmented
                        label={said.body}
                        size="sm"
                        value={query.body === "" ? "any" : query.body}
                        options={bodyOptions}
                        onValueChange={(next) => {
                            onChange({ ...query, body: next === "any" ? "" : next });
                        }}
                    />
                </div>
                <Checkbox
                    checked={query.findings}
                    label={said.findings}
                    onChange={(event) => {
                        onChange({ ...query, findings: event.target.checked });
                    }}
                />
            </div>

            <div className="flex flex-col gap-1.5">
                <SectionLabel>{said.branch}</SectionLabel>
                <BranchFilter siteId={siteId} query={query} onChange={onChange} />
            </div>

            <div className="flex flex-col gap-1.5">
                <SectionLabel>{said.prefix}</SectionLabel>
                <PrefixFilter query={query} onChange={onChange} />
            </div>

            <div className="flex flex-col gap-1.5">
                <SectionLabel>{copy.pages.filters.entity}</SectionLabel>
                <div title={index.entities.length === 0 ? copy.pages.filters.noEntities : undefined}>
                    <Select
                        value={query.entityId === "" ? anyEntity : query.entityId}
                        options={entityOptions}
                        aria-label={copy.pages.filters.entity}
                        disabled={index.entities.length === 0}
                        onValueChange={(next) => {
                            onChange({
                                ...query,
                                entityId: next === anyEntity ? "" : next,
                                descendants: next === anyEntity ? false : query.descendants,
                                unmapped: false,
                            });
                        }}
                    />
                </div>
                <Checkbox
                    checked={query.descendants}
                    disabled={query.entityId === ""}
                    label={copy.pages.filters.descendants}
                    onChange={(event) => {
                        onChange({ ...query, descendants: event.target.checked });
                    }}
                />
                <div className="flex items-center justify-between gap-2">
                    <Checkbox
                        checked={query.unmapped}
                        label={copy.pages.filters.unmapped}
                        onChange={(event) => {
                            onChange({
                                ...query,
                                unmapped: event.target.checked,
                                entityId: event.target.checked ? "" : query.entityId,
                                descendants: event.target.checked ? false : query.descendants,
                            });
                        }}
                    />
                    <span className="shrink-0 font-mono text-2xs text-ink-faint">{totals?.unmapped ?? ""}</span>
                </div>
            </div>
        </div>
    );
}
