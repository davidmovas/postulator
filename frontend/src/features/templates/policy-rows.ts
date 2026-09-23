import { useMemo } from "react";

import { flatten } from "../../data/call.js";
import { useEffectivePolicy, usePolicies } from "../../data/hooks/templates.js";
import type { LinkPolicy } from "../../data/types.js";

const pageSize = 100;

export interface PolicyRows {
    shown: readonly LinkPolicy[];
    inForceId: string | null;
    pending: boolean;
    error: unknown;
    hasMore: boolean;
    loadingMore: boolean;
    loadMore: () => void;
    retry: () => void;
}

export function usePolicyRows(siteId: string): PolicyRows {
    const globals = usePolicies({ scope: "global" }, null, pageSize);
    const locals = usePolicies({ siteId }, null, pageSize);
    const effective = useEffectivePolicy(siteId === "" ? null : siteId);
    const globalRows = useMemo(() => flatten(globals.data?.pages), [globals.data]);
    const localRows = useMemo(() => flatten(locals.data?.pages), [locals.data]);
    const shown = useMemo(() => [...globalRows, ...localRows], [globalRows, localRows]);

    return {
        shown,
        inForceId: effective.data?.policy.id ?? null,
        pending: globals.isPending || locals.isPending,
        error: globals.error ?? locals.error,
        hasMore: globals.hasNextPage || locals.hasNextPage,
        loadingMore: globals.isFetchingNextPage || locals.isFetchingNextPage,
        loadMore: () => {
            if (globals.hasNextPage) {
                void globals.fetchNextPage();
            }
            if (locals.hasNextPage) {
                void locals.fetchNextPage();
            }
        },
        retry: () => {
            void globals.refetch();
            void locals.refetch();
        },
    };
}
