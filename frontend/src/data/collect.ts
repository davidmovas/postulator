import type { Cursor, List } from "../lib/paging.js";
import { maxLimit } from "../lib/paging.js";
import type { PageArgs } from "./call.js";

export type PageOf<F, Item> = (args: PageArgs<F>) => Promise<List<Item>>;

export async function collectAll<F, Item>(fetch: PageOf<F, Item>, filters: F): Promise<Item[]> {
    const out: Item[] = [];
    let cursor: Cursor | undefined;
    for (;;) {
        const asked: PageArgs<F> = cursor === undefined ? { ...filters, limit: maxLimit } : { ...filters, limit: maxLimit, cursor };
        const page = await fetch(asked);
        out.push(...page.items);
        const next = page.hasMore ? page.nextCursor : undefined;
        if (next === undefined || next === cursor) {
            return out;
        }
        cursor = next;
    }
}
