import { useEffect, useMemo } from "react";

import { useSiteMap } from "../../data/hooks/pages.js";
import type { SiteMapFilter, SiteMapNode, SiteMapView } from "../../data/types.js";
import type { Branch } from "./model/tree.js";

export interface Loaded {
    branch: Branch;
    view: SiteMapView | null;
    error: unknown;
    more: () => void;
}

export function useBranch(filter: SiteMapFilter, enabled = true): Loaded {
    const listed = useSiteMap(filter, enabled);
    const pages = listed.data?.pages;
    const nodes = useMemo<readonly SiteMapNode[]>(() => (pages ?? []).flatMap((view) => view.nodes.items ?? []), [pages]);
    const { hasNextPage, isPending, isError, isFetchingNextPage, fetchNextPage, error } = listed;
    const view = pages?.[0] ?? null;
    return useMemo(
        () => ({
            branch: { nodes, hasMore: hasNextPage, pending: isPending && !isError },
            view,
            error,
            more: () => {
                if (!isFetchingNextPage) {
                    void fetchNextPage();
                }
            },
        }),
        [nodes, hasNextPage, isPending, isError, view, error, isFetchingNextPage, fetchNextPage],
    );
}

export interface BranchLoaderProps {
    siteId: string;
    parentId: string;
    onLoaded: (parentId: string, loaded: Loaded) => void;
}

export function BranchLoader({ siteId, parentId, onLoaded }: BranchLoaderProps): null {
    const loaded = useBranch(parentId === "" ? { siteId } : { siteId, parentId });

    useEffect(() => {
        onLoaded(parentId, loaded);
    }, [parentId, loaded, onLoaded]);

    return null;
}
