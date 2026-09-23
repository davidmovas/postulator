import { describe, expect, it } from "vitest";

import type { Conversation } from "../../../data/types.js";
import { groupBySite, matches } from "./conversations.js";

function conversation(id: string, title: string, siteId: string | null, createdAt: string): Conversation {
    return { id, siteId, title, mode: "confirm", createdAt, updatedAt: createdAt };
}

const rows = [
    conversation("c1", "Plan the hub", "s1", "2026-09-19T10:00:00Z"),
    conversation("c2", "", null, "2026-09-19T11:00:00Z"),
    conversation("c3", "Fix the links", "s2", "2026-09-19T12:00:00Z"),
    conversation("c4", "Audit mugs", "s1", "2026-09-19T13:00:00Z"),
];

const siteNames = new Map([
    ["s1", "Clay and Kiln"],
    ["s2", "Mugs Direct"],
]);

describe("groupBySite", () => {
    it("puts the current site first, other sites by newest conversation, and the siteless ones last", () => {
        const groups = groupBySite(rows, siteNames, "s2");
        expect(groups.map((group) => group.key)).toStrictEqual(["s2", "s1", "global"]);
        expect(groups[1]?.name).toBe("Clay and Kiln");
        expect(groups[1]?.conversations.map((held) => held.id)).toStrictEqual(["c4", "c1"]);
        expect(groups[2]?.name).toBeNull();
    });

    it("names a site that is not in the list by its id", () => {
        const groups = groupBySite(rows, new Map(), null);
        expect(groups.find((group) => group.key === "s1")?.name).toBe("s1");
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
