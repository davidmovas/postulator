import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { usePageTypes } from "../../data/hooks/templates.js";
import type { SelectOption } from "../../ui/index.js";
import { Input, Select, Toolbar, useDebounced } from "../../ui/index.js";
import type { MapQuery } from "./model/params.js";

const said = copy.map;

const everyType = "every";
const searchDelayMs = 250;

export interface MapToolbarProps {
    siteId: string;
    query: MapQuery;
    onChange: (next: MapQuery) => void;
}

export function MapToolbar({ siteId, query, onChange }: MapToolbarProps): ReactElement {
    const types = usePageTypes(siteId);
    const [typed, setTyped] = useState(query.search);
    const settled = useDebounced(typed, searchDelayMs);

    useEffect(() => {
        setTyped(query.search);
    }, [query.search]);

    useEffect(() => {
        if (settled.trim() !== query.search.trim()) {
            onChange({ ...query, search: settled.trim() });
        }
    }, [settled]);

    const typeOptions: SelectOption<string>[] = [
        { value: everyType, label: said.everyType },
        ...(types.data?.types ?? []).map((type) => ({ value: type.id, label: type.name })),
    ];

    return (
        <Toolbar label={said.title}>
            <div className="w-64">
                <Input
                    type="search"
                    value={typed}
                    placeholder={said.searchHint}
                    aria-label={said.search}
                    onChange={(event) => {
                        setTyped(event.target.value);
                    }}
                />
            </div>
            <div className="w-44">
                <Select
                    size="sm"
                    value={query.type === "" ? everyType : query.type}
                    options={typeOptions}
                    aria-label={said.type}
                    onValueChange={(next) => {
                        onChange({ ...query, type: next === everyType ? "" : next });
                    }}
                />
            </div>
        </Toolbar>
    );
}
