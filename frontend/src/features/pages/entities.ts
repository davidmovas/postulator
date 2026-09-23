import { useEffect, useMemo } from "react";

import { flatten } from "../../data/call.js";
import { useEntities } from "../../data/hooks/graph.js";
import type { Entity } from "../../data/types.js";

const entityPageSize = 500;
const maxEntityPages = 4;

export interface EntityIndex {
    entities: readonly Entity[];
    byId: ReadonlyMap<string, Entity>;
    complete: boolean;
    loading: boolean;
}

export function useEntityIndex(siteId: string): EntityIndex {
    const query = useEntities({ siteId }, { field: "name", desc: false }, entityPageSize);
    const { data, hasNextPage, isFetchingNextPage, fetchNextPage, isPending } = query;
    const loadedPages = data?.pages.length ?? 0;

    useEffect(() => {
        if (hasNextPage && !isFetchingNextPage && loadedPages > 0 && loadedPages < maxEntityPages) {
            void fetchNextPage();
        }
    }, [hasNextPage, isFetchingNextPage, loadedPages, fetchNextPage]);

    const entities = useMemo(() => flatten(data?.pages), [data]);
    const byId = useMemo(() => {
        const index = new Map<string, Entity>();
        for (const entity of entities) {
            index.set(entity.id, entity);
        }
        return index;
    }, [entities]);

    return { entities, byId, complete: !hasNextPage, loading: isPending };
}
