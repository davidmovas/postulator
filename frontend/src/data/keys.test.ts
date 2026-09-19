import { describe, expect, test } from "vitest";

import { defaultLimit, maxLimit } from "../lib/paging.js";
import { keys, siteOf } from "./keys.js";

function isPrefix(prefix: readonly unknown[], key: readonly unknown[]): boolean {
    return (
        prefix.length <= key.length &&
        prefix.every((segment, index) => JSON.stringify(segment) === JSON.stringify(key[index]))
    );
}

describe("the limit is part of a list key", () => {
    const filter = { siteId: "s1" };
    const sort = { field: "path", desc: false };

    test("two limits over the same filter and sort do not share a cache entry", () => {
        expect(keys.pages.list(filter, sort, 1)).not.toEqual(keys.pages.list(filter, sort, 200));
    });

    test("an omitted limit keys the same entry as the limit that goes on the wire", () => {
        expect(keys.pages.list(filter, sort)).toEqual(keys.pages.list(filter, sort, defaultLimit));
    });

    test("a limit over the ceiling keys the same entry as the ceiling", () => {
        expect(keys.pages.list(filter, sort, maxLimit + 1)).toEqual(keys.pages.list(filter, sort, maxLimit));
    });

    test("the broad handle still prefixes every limit", () => {
        expect(isPrefix(keys.pages.lists(), keys.pages.list(filter, sort, 1))).toBe(true);
        expect(isPrefix(keys.sites.lists(), keys.sites.list({}, null, 25))).toBe(true);
        expect(isPrefix(keys.runs.lists(), keys.runs.list({}, null, 25))).toBe(true);
        expect(isPrefix(keys.templates.lists(), keys.templates.list({}, null, 25))).toBe(true);
        expect(isPrefix(keys.policies.lists(), keys.policies.list({}, null, 25))).toBe(true);
        expect(isPrefix(keys.schedules.lists(), keys.schedules.list({}, 25))).toBe(true);
        expect(isPrefix(keys.graph.entityLists(), keys.graph.entities(filter, null, 25))).toBe(true);
        expect(isPrefix(keys.graph.edgeLists(), keys.graph.edges(filter, 25))).toBe(true);
        expect(isPrefix(keys.runs.itemsOf("r1"), keys.runs.items("r1", "failed", 25))).toBe(true);
        expect(isPrefix(keys.agent.conversationsAll(), keys.agent.conversations({}, 25))).toBe(true);
        expect(isPrefix(keys.agent.pendingAll(), keys.agent.pending({}, 25))).toBe(true);
    });

    test("the sort still separates two keys, whatever the limit", () => {
        expect(keys.pages.list(filter, sort, 25)).not.toEqual(
            keys.pages.list(filter, { field: "path", desc: true }, 25),
        );
    });

    test("a site-scoped event still reads the filter out of the key", () => {
        expect(siteOf(keys.pages.list(filter, sort, 1))).toBe("s1");
        expect(siteOf(keys.graph.edges(filter, 1))).toBe("s1");
    });
});

describe("the conversation handle invalidates every message page", () => {
    test("messagesOf prefixes the keyed message list whatever its limit", () => {
        const filter = { conversationId: "c1" };
        expect(isPrefix(keys.agent.messagesOf("c1"), keys.agent.messages(filter, 25))).toBe(true);
        expect(isPrefix(keys.agent.messagesOf("c1"), keys.agent.messages(filter))).toBe(true);
        expect(isPrefix(keys.agent.messagesOf("c2"), keys.agent.messages(filter, 25))).toBe(false);
    });
});
