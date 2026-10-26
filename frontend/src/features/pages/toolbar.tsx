import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { Input, SearchIcon, Segmented, Toolbar } from "../../ui/index.js";
import type { PagesQuery } from "./params.js";
import type { PagesView } from "./views/memory.js";
import { pagesViews } from "./views/memory.js";

const said = copy.pages;

const viewOptions: readonly SegmentedOption<PagesView>[] = pagesViews.map((view) => ({ value: view, label: said.views[view] }));

export interface PageToolbarProps {
    view: PagesView;
    onView: (view: PagesView) => void;
    query: PagesQuery;
    onChange: (next: PagesQuery) => void;
}

export function PageToolbar({ view, onView, query, onChange }: PageToolbarProps): ReactElement {
    const [search, setSearch] = useState(query.search);

    useEffect(() => {
        setSearch(query.search);
    }, [query.search]);

    const apply = (value: string): void => {
        const trimmed = value.trim();
        if (trimmed !== query.search) {
            onChange({ ...query, search: trimmed });
        }
    };

    return (
        <Toolbar label={said.title}>
            <Segmented label={said.views.label} value={view} options={viewOptions} onValueChange={onView} />
            {view === "list" ? (
                <div className="relative w-64">
                    <SearchIcon size={14} className="pointer-events-none absolute top-1/2 left-2 -translate-y-1/2 text-ink-faint" />
                    <Input
                        type="search"
                        value={search}
                        placeholder={said.searchWords}
                        aria-label={said.searchWords}
                        className="pl-7"
                        onChange={(event) => {
                            setSearch(event.target.value);
                            if (event.target.value === "") {
                                apply("");
                            }
                        }}
                        onBlur={() => {
                            apply(search);
                        }}
                        onKeyDown={(event) => {
                            if (event.key === "Enter") {
                                apply(search);
                            }
                        }}
                    />
                </div>
            ) : null}
        </Toolbar>
    );
}
