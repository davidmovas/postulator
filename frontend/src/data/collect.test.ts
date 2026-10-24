import { describe, expect, it } from "vitest";

import type { List } from "../lib/paging.js";
import { maxLimit } from "../lib/paging.js";
import type { PageArgs } from "./call.js";
import { collectAll } from "./collect.js";

interface Filter {
    siteId: string;
}

function pages(lists: readonly List<string>[]) {
    const asked: PageArgs<Filter>[] = [];
    const fetch = (args: PageArgs<Filter>): Promise<List<string>> => {
        asked.push(args);
        return Promise.resolve(lists[asked.length - 1] ?? { items: [], hasMore: false });
    };
    return { asked, fetch };
}

describe("collecting every item of a cursored list", () => {
    it("follows the cursor to the last page at the widest page the site serves", async () => {
        const { asked, fetch } = pages([
            { items: ["a", "b"], nextCursor: "c1", hasMore: true },
            { items: ["c"], nextCursor: "c2", hasMore: true },
            { items: ["d"], hasMore: false },
        ]);

        expect(await collectAll(fetch, { siteId: "s1" })).toStrictEqual(["a", "b", "c", "d"]);
        expect(asked).toStrictEqual([
            { siteId: "s1", limit: maxLimit },
            { siteId: "s1", limit: maxLimit, cursor: "c1" },
            { siteId: "s1", limit: maxLimit, cursor: "c2" },
        ]);
    });

    it("stops on a page that says more but names no cursor, or the cursor it was given", async () => {
        const nameless = pages([{ items: ["a"], hasMore: true }]);
        expect(await collectAll(nameless.fetch, { siteId: "s1" })).toStrictEqual(["a"]);

        const stuck = pages([
            { items: ["a"], nextCursor: "c1", hasMore: true },
            { items: ["b"], nextCursor: "c1", hasMore: true },
        ]);
        expect(await collectAll(stuck.fetch, { siteId: "s1" })).toStrictEqual(["a", "b"]);
        expect(stuck.asked).toHaveLength(2);
    });
});
