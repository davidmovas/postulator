import { describe, expect, test } from "vitest";

import type { Message } from "../types.js";
import { answeredBy } from "./reconcile.js";

function message(id: string, role: string, createdAt: string | null): Message {
    return { id, conversationId: "c1", seq: 1, role, text: "", createdAt };
}

describe("answeredBy decides whether the turn produced a message", () => {
    const startedAt = Date.parse("2026-09-20T10:00:00Z");

    test("the assistant message the turn announced counts", () => {
        const messages = [message("u1", "user", null), message("a1", "assistant", null)];
        expect(answeredBy(messages, "a1", startedAt)).toBe(true);
    });

    test("another assistant message does not stand in for the one that was promised", () => {
        expect(answeredBy([message("a0", "assistant", null)], "a1", startedAt)).toBe(false);
    });

    test("with no assistant id an answer written after the turn started counts", () => {
        const messages = [message("a0", "assistant", "2026-09-20T10:00:04Z")];
        expect(answeredBy(messages, null, startedAt)).toBe(true);
    });

    test("with no assistant id an older answer does not count", () => {
        const messages = [message("a0", "assistant", "2026-09-20T09:58:00Z")];
        expect(answeredBy(messages, null, startedAt)).toBe(false);
    });

    test("a tool row is never an answer", () => {
        expect(answeredBy([message("t1", "tool", "2026-09-20T10:00:04Z")], null, startedAt)).toBe(false);
    });
});
