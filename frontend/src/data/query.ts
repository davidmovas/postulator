import type {
    InfiniteData,
    QueryKey,
    UseInfiniteQueryResult,
    UseQueryOptions,
    UseQueryResult,
} from "@tanstack/react-query";
import { keepPreviousData, useInfiniteQuery, useQueries, useQuery } from "@tanstack/react-query";

import type { Cursor, List, Sort } from "../lib/paging.js";
import type { Paged } from "./call.js";
import { infiniteGcTimeMs, quietMeta } from "./client.js";
import type { Code } from "./errors.js";
import { useLockGate } from "./lock.js";

export type UnlockedQueryOptions<TQueryFnData, TData> = Omit<
    UseQueryOptions<TQueryFnData, Error, TData, QueryKey>,
    "enabled" | "meta"
> & { enabled?: boolean; quiet?: readonly Code[] };

export function useUnlockedQuery<TQueryFnData, TData = TQueryFnData>(
    options: UnlockedQueryOptions<TQueryFnData, TData>,
): UseQueryResult<TData, Error> {
    const gate = useLockGate();
    const { quiet, ...rest } = options;
    const enabled = (options.enabled ?? true) && gate.ready && !gate.locked;
    return useQuery({ ...rest, enabled, meta: quiet === undefined ? undefined : quietMeta(quiet) });
}

export function useUnlockedQueries<TQueryFnData>(
    options: readonly UnlockedQueryOptions<TQueryFnData, TQueryFnData>[],
): UseQueryResult<TQueryFnData, Error>[] {
    const gate = useLockGate();
    const ready = gate.ready && !gate.locked;
    return useQueries({
        queries: options.map(({ quiet, ...rest }) => ({
            ...rest,
            enabled: (rest.enabled ?? true) && ready,
            meta: quiet === undefined ? undefined : quietMeta(quiet),
        })),
    });
}

export function nextPageParam<T>(last: List<T>): Cursor | undefined {
    return last.hasMore && last.nextCursor !== undefined ? last.nextCursor : undefined;
}

export interface UnlockedInfiniteOptions<F, Item> {
    queryKey: QueryKey;
    fetch: Paged<F, Item>;
    filters: F;
    sort?: Sort | null;
    limit?: number;
    enabled?: boolean;
    quiet?: readonly Code[];
}

export function useUnlockedInfinite<F, Item>(
    options: UnlockedInfiniteOptions<F, Item>,
): UseInfiniteQueryResult<InfiniteData<List<Item>, Cursor | undefined>, Error> {
    const gate = useLockGate();
    const enabled = (options.enabled ?? true) && gate.ready && !gate.locked;
    return useInfiniteQuery({
        queryKey: options.queryKey,
        enabled,
        meta: options.quiet === undefined ? undefined : quietMeta(options.quiet),
        initialPageParam: undefined as Cursor | undefined,
        queryFn: ({ pageParam, signal }) =>
            options.fetch(
                { ...options.filters, cursor: pageParam, limit: options.limit, sort: options.sort ?? null },
                signal,
            ) as Promise<List<Item>>,
        getNextPageParam: nextPageParam,
        getPreviousPageParam: () => undefined,
        refetchOnWindowFocus: false,
        gcTime: infiniteGcTimeMs,
        placeholderData: keepPreviousData,
    });
}
