export type Cursor = string;

export interface Sort {
    field: string;
    desc: boolean;
}

export interface ListRequest {
    cursor?: Cursor;
    limit: number;
    sort?: Sort | null;
}

export interface List<T> {
    items: T[];
    nextCursor?: Cursor;
    prevCursor?: Cursor;
    hasMore: boolean;
}

export const defaultLimit = 50;
export const maxLimit = 500;

export function page(limit: number, cursor?: Cursor): ListRequest {
    const clamped = limit <= 0 ? defaultLimit : Math.min(limit, maxLimit);
    return cursor === undefined ? { limit: clamped } : { cursor, limit: clamped };
}

export function listOf<T>(raw: { items: unknown; nextCursor?: Cursor; prevCursor?: Cursor; hasMore: boolean }): List<T> {
    return {
        items: (raw.items ?? []) as T[],
        nextCursor: raw.nextCursor,
        prevCursor: raw.prevCursor,
        hasMore: raw.hasMore,
    };
}
