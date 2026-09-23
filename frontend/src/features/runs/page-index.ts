import { useEffect, useMemo } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { usePages } from "../../data/hooks/pages.js";
import type { Page } from "../../data/types.js";

const pageSize = 500;
const maxPages = 4;

export interface PageIndex {
    byId: ReadonlyMap<string, Page>;
    complete: boolean;
    loading: boolean;
}

export function usePageIndex(siteId: string): PageIndex {
    const query = usePages({ siteId }, { field: "path", desc: false }, pageSize);
    const { data, hasNextPage, isFetchingNextPage, fetchNextPage, isPending } = query;
    const loadedPages = data?.pages.length ?? 0;

    useEffect(() => {
        if (hasNextPage && !isFetchingNextPage && loadedPages > 0 && loadedPages < maxPages) {
            void fetchNextPage();
        }
    }, [hasNextPage, isFetchingNextPage, loadedPages, fetchNextPage]);

    const byId = useMemo(() => {
        const index = new Map<string, Page>();
        for (const page of flatten(data?.pages)) {
            index.set(page.id, page);
        }
        return index;
    }, [data]);

    return { byId, complete: !hasNextPage, loading: isPending };
}

export function pathOf(index: PageIndex, pageId: string): string {
    const held = index.byId.get(pageId)?.path;
    if (held !== undefined) {
        return held;
    }
    return index.loading ? copy.app.loading : copy.runs.wholeSite;
}
