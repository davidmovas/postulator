import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import type { PageSort } from "../../data/sorts.js";
import type { SelectOption } from "../../ui/index.js";
import { Button, Input, Select, Toolbar, UnfoldLessIcon, UnfoldMoreIcon } from "../../ui/index.js";
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

export interface TreeControls {
    nodes: number;
    onExpandAll: () => void;
    onCollapseAll: () => void;
}

export interface PageToolbarProps {
    query: PagesQuery;
    disabled: boolean;
    tree: TreeControls | null;
    onChange: (next: PagesQuery) => void;
}

export function PageToolbar({ query, disabled, tree, onChange }: PageToolbarProps): ReactElement {
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
            <div className="w-64" title={disabled ? copy.pages.tableOnly : undefined}>
                <Input
                    type="search"
                    mono={true}
                    value={prefix}
                    placeholder="/"
                    disabled={disabled}
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
            <div className="w-40" title={disabled ? copy.pages.tableOnly : undefined}>
                <Select
                    value={sortChoice(query.sort)}
                    options={sortOptions}
                    disabled={disabled}
                    aria-label={copy.pages.sort}
                    onValueChange={(next) => {
                        onChange({ ...query, sort: sorts[next] });
                    }}
                />
            </div>
            {tree === null ? null : (
                <div className="ml-auto flex shrink-0 items-center gap-2">
                    <span className="font-mono text-2xs text-ink-faint">{copy.pages.tree.nodes(tree.nodes)}</span>
                    <Button size="sm" variant="ghost" icon={UnfoldMoreIcon} onClick={tree.onExpandAll}>
                        {copy.pages.tree.expandAll}
                    </Button>
                    <Button size="sm" variant="ghost" icon={UnfoldLessIcon} onClick={tree.onCollapseAll}>
                        {copy.pages.tree.collapseAll}
                    </Button>
                </div>
            )}
        </Toolbar>
    );
}
