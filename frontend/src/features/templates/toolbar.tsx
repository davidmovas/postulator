import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import type { SegmentedOption, SelectOption } from "../../ui/index.js";
import { Input, Segmented, Select, Toolbar } from "../../ui/index.js";
import { pageKindLabel } from "./labels.js";
import type { ScopeFilter, SortChoice, TemplatesQuery } from "./params.js";
import { sortChoice, sorts } from "./params.js";

const scopeOptions: readonly SegmentedOption<ScopeFilter>[] = [
    { value: "all", label: copy.templates.filters.scopeAll },
    { value: "global", label: copy.templates.scope.global },
    { value: "site", label: copy.templates.scope.site },
];

const sortOptions: readonly SelectOption<SortChoice>[] = [
    { value: "nameAsc", label: copy.templates.sortOptions.nameAsc },
    { value: "nameDesc", label: copy.templates.sortOptions.nameDesc },
    { value: "newest", label: copy.templates.sortOptions.newest },
    { value: "oldest", label: copy.templates.sortOptions.oldest },
];

const anyKind = "*";

export interface TemplateToolbarProps {
    query: TemplatesQuery;
    kinds: readonly string[];
    onChange: (next: TemplatesQuery) => void;
}

export function TemplateToolbar({ query, kinds, onChange }: TemplateToolbarProps): ReactElement {
    const [search, setSearch] = useState(query.search);

    useEffect(() => {
        setSearch(query.search);
    }, [query.search]);

    const apply = (): void => {
        const trimmed = search.trim();
        if (trimmed !== query.search) {
            onChange({ ...query, search: trimmed });
        }
    };

    const kindOptions: readonly SelectOption<string>[] = [
        { value: anyKind, label: copy.templates.filters.anyPageKind },
        ...kinds.map((kind) => ({ value: kind, label: pageKindLabel(kind) })),
    ];

    return (
        <Toolbar label={copy.templates.title}>
            <div className="w-56" title={copy.templates.searchLoaded}>
                <Input
                    type="search"
                    value={search}
                    placeholder={copy.templates.search}
                    aria-label={copy.templates.search}
                    data-template-search={true}
                    onChange={(event) => {
                        setSearch(event.target.value);
                    }}
                    onBlur={apply}
                    onKeyDown={(event) => {
                        if (event.key === "Enter") {
                            apply();
                        }
                    }}
                />
            </div>
            <Segmented
                label={copy.templates.filters.scope}
                value={query.scope}
                options={scopeOptions}
                onValueChange={(scope) => {
                    onChange({ ...query, scope });
                }}
            />
            <div className="w-40">
                <Select
                    value={query.pageKind === "" ? anyKind : query.pageKind}
                    options={kindOptions}
                    aria-label={copy.templates.filters.pageKind}
                    onValueChange={(pageKind) => {
                        onChange({ ...query, pageKind: pageKind === anyKind ? "" : pageKind });
                    }}
                />
            </div>
            <div className="ml-auto w-40">
                <Select
                    value={sortChoice(query.sort)}
                    options={sortOptions}
                    aria-label={copy.templates.filters.sort}
                    onValueChange={(next) => {
                        onChange({ ...query, sort: sorts[next] });
                    }}
                />
            </div>
        </Toolbar>
    );
}
