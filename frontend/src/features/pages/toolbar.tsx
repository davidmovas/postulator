import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import type { PageSort } from "../../data/sorts.js";
import type { SelectOption } from "../../ui/index.js";
import { Input, Select, Toolbar } from "../../ui/index.js";
import type { PagesQuery } from "./params.js";

type SortChoice = "pathAsc" | "pathDesc" | "newest" | "oldest";

const sortOptions: readonly SelectOption<SortChoice>[] = [
    { value: "pathAsc", label: copy.pages.sortOptions.pathAsc },
    { value: "pathDesc", label: copy.pages.sortOptions.pathDesc },
    { value: "newest", label: copy.pages.sortOptions.newest },
    { value: "oldest", label: copy.pages.sortOptions.oldest },
];

const sorts: Readonly<Record<SortChoice, PageSort>> = {
    pathAsc: { field: "path", desc: false },
    pathDesc: { field: "path", desc: true },
    newest: { field: "createdAt", desc: true },
    oldest: { field: "createdAt", desc: false },
};

export function sortChoice(sort: PageSort | null): SortChoice {
    if (sort === null) {
        return "pathAsc";
    }
    if (sort.field === "createdAt") {
        return sort.desc ? "newest" : "oldest";
    }
    return sort.desc ? "pathDesc" : "pathAsc";
}

export interface PageToolbarProps {
    query: PagesQuery;
    onChange: (next: PagesQuery) => void;
}

export function PageToolbar({ query, onChange }: PageToolbarProps): ReactElement {
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
        <Toolbar label={copy.pages.title}>
            <div className="w-64">
                <Input
                    type="search"
                    mono={true}
                    value={prefix}
                    placeholder="/"
                    aria-label={copy.pages.search}
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
            </div>
            <div className="w-40">
                <Select
                    value={sortChoice(query.sort)}
                    options={sortOptions}
                    aria-label={copy.pages.sort}
                    onValueChange={(next) => {
                        onChange({ ...query, sort: sorts[next] });
                    }}
                />
            </div>
        </Toolbar>
    );
}
