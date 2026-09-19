import { describe, expect, it } from "vitest";

import type { Conversation, PendingAction } from "../../../data/types.js";
import { groupActions, statusCounts } from "./inbox.js";

function action(id: string, conversationId: string, createdAt: string, status = "pending"): PendingAction {
    return { id, conversationId, tool: "pages_delete", args: {}, summary: "x", status, createdAt, updatedAt: createdAt };
}

function conversation(id: string, title: string, siteId: string | null): Conversation {
    return { id, siteId, title, mode: "confirm", createdAt: "2026-09-19T09:00:00Z", updatedAt: "2026-09-19T09:00:00Z" };
}

const conversations = [conversation("c1", "Plan the hub", "s1"), conversation("c2", "", null)];

describe("groupActions", () => {
    it("groups actions under their conversation, newest group first, actions oldest first", () => {
        const groups = groupActions(
            [
                action("a3", "c2", "2026-09-19T12:00:00Z"),
                action("a1", "c1", "2026-09-19T10:00:00Z"),
                action("a2", "c1", "2026-09-19T11:00:00Z"),
            ],
            conversations,
        );
        expect(groups.map((group) => group.conversationId)).toStrictEqual(["c2", "c1"]);
        expect(groups[1]?.actions.map((held) => held.id)).toStrictEqual(["a1", "a2"]);
        expect(groups[0]?.title).toBe("");
        expect(groups[1]?.title).toBe("Plan the hub");
        expect(groups[1]?.siteId).toBe("s1");
    });

    it("keeps an action whose conversation is gone in a group of its own", () => {
        const groups = groupActions([action("a1", "gone", "2026-09-19T10:00:00Z")], conversations);
        expect(groups).toHaveLength(1);
        expect(groups[0]).toMatchObject({ conversationId: "gone", title: null, siteId: null });
    });

    it("flattens back into keyboard order", () => {
        const groups = groupActions(
            [action("a1", "c1", "2026-09-19T10:00:00Z"), action("a2", "c2", "2026-09-19T11:00:00Z")],
            conversations,
        );
        expect(groups.flatMap((group) => group.actions.map((held) => held.id))).toStrictEqual(["a2", "a1"]);
    });
});

describe("statusCounts", () => {
    it("counts every status the tabs show", () => {
        const counts = statusCounts([
            action("a1", "c1", "t"),
            action("a2", "c1", "t", "executed"),
            action("a3", "c1", "t", "rejected"),
            action("a4", "c1", "t", "failed"),
            action("a5", "c1", "t", "approved"),
        ]);
        expect(counts).toStrictEqual({ pending: 1, executed: 1, rejected: 1, failed: 1 });
    });
});
