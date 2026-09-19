import { describe, expect, test } from "vitest";

import type { List } from "../lib/paging.js";
import { nextPageParam } from "./query.js";

function listOf(hasMore: boolean, nextCursor?: string): List<string> {
    return { items: ["a"], hasMore, nextCursor };
}

describe("cursor paging", () => {
    test("follows the cursor while the server says there is more", () => {
        expect(nextPageParam(listOf(true, "eyJ4Ijoxfq"))).toBe("eyJ4Ijoxfq");
    });

    test("stops when hasMore is true but no cursor was issued", () => {
        expect(nextPageParam(listOf(true))).toBeUndefined();
    });

    test("stops on the last page even when a cursor is present", () => {
        expect(nextPageParam(listOf(false, "eyJ4Ijoxfq"))).toBeUndefined();
    });

    test("returns undefined rather than null so the query stops", () => {
        const stop = nextPageParam(listOf(false));
        expect(stop).toBeUndefined();
        expect(stop).not.toBeNull();
    });
});
