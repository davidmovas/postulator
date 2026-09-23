import type { CancellablePromise } from "@wailsio/runtime";

import type { Cursor, List, ListRequest, Sort } from "../lib/paging.js";
import { defaultLimit, listOf, maxLimit } from "../lib/paging.js";
import type { Wire } from "./wire.js";

export type Bound<Req, Res> = (req: Req) => CancellablePromise<Res>;

export type Typed<Req, Res> = (req: Req, signal?: AbortSignal) => Promise<Wire<Res>>;

export type PageArgs<F> = F & { cursor?: Cursor; limit?: number; sort?: Sort | null };

export type Paged<F, Item> = (args: PageArgs<F>, signal?: AbortSignal) => Promise<List<Wire<Item>>>;

interface RawList {
    items: unknown;
    nextCursor?: Cursor;
    prevCursor?: Cursor;
    hasMore: boolean;
}

export function clampLimit(limit: number | undefined): number {
    return limit === undefined || limit <= 0 ? defaultLimit : Math.min(limit, maxLimit);
}

function settle<Res>(pending: CancellablePromise<Res>, signal?: AbortSignal): Promise<Res> {
    return signal === undefined ? pending : pending.cancelOn(signal);
}

export function one<Req, Res>(fn: Bound<Req, Res>): Typed<Req, Res> {
    return async (req: Req, signal?: AbortSignal): Promise<Wire<Res>> => {
        const answered = await settle(fn(req), signal);
        return answered as unknown as Wire<Res>;
    };
}

export function listed<Req, Item>(fn: Bound<Req, unknown>): Paged<Omit<Req, keyof ListRequest>, Item> {
    return async (
        args: PageArgs<Omit<Req, keyof ListRequest>>,
        signal?: AbortSignal,
    ): Promise<List<Wire<Item>>> => {
        const { cursor, limit, sort, ...filters } = args;
        const clamped = clampLimit(limit);
        const request: ListRequest & Record<string, unknown> = { ...filters, limit: clamped };
        if (cursor !== undefined) {
            request.cursor = cursor;
        }
        if (sort !== undefined && sort !== null) {
            request.sort = sort;
        }
        const answered = await settle(fn(request as unknown as Req), signal);
        return listOf<Wire<Item>>(answered as RawList);
    };
}

export function isCancellation(thrown: unknown): boolean {
    if (typeof thrown !== "object" || thrown === null || !("name" in thrown)) {
        return false;
    }
    return (thrown as { name: unknown }).name === "CancelError";
}

export function flatten<T>(pages: readonly List<T>[] | undefined): T[] {
    if (pages === undefined) {
        return [];
    }
    const out: T[] = [];
    for (const loaded of pages) {
        out.push(...loaded.items);
    }
    return out;
}
