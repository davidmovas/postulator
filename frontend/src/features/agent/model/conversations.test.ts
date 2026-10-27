import { describe, expect, it } from "vitest";

import type { Conversation } from "../../../data/types.js";
import { inScope, matches } from "./conversations.js";

function conversation(id: string, title: string, siteId: string | null, createdAt: string): Conversation {
    return { id, siteId, title, mode: "confirm", createdAt, updatedAt: createdAt };
}

const rows = [
    conversation("c1", "Plan the hub", "s1", "2026-09-19T10:00:00Z"),
    conversation("c2", "", null, "2026-09-19T11:00:00Z"),
    conversation("c3", "Fix the links", "s2", "2026-09-19T12:00:00Z"),
    conversation("c4", "Audit mugs", "s1", "2026-09-19T13:00:00Z"),
    conversation("c5", "Add a site", null, "2026-09-19T14:00:00Z"),
];

describe("inScope", () => {
    it.each([
        { scope: "s1", query: "", ids: ["c4", "c1"] },
        { scope: "s2", query: "", ids: ["c3"] },
        { scope: null, query: "", ids: ["c5", "c2"] },
        { scope: "s1", query: "hub", ids: ["c1"] },
        { scope: null, query: "site", ids: ["c5"] },
        { scope: "s3", query: "", ids: [] },
    ])("keeps the conversations of $scope matching '$query', newest first", ({ scope, query, ids }) => {
        expect(inScope(rows, scope, query).map((held) => held.id)).toStrictEqual(ids);
    });
});

describe("matches", () => {
    it("matches on the title, case insensitive, and an untitled conversation matches nothing but the empty query", () => {
        expect(matches(rows[0] as Conversation, "hub")).toBe(true);
        expect(matches(rows[0] as Conversation, "HUB")).toBe(true);
        expect(matches(rows[1] as Conversation, "hub")).toBe(false);
        expect(matches(rows[1] as Conversation, "")).toBe(true);
    });
});
