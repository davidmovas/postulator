import { afterEach, describe, expect, test } from "vitest";

import {
    applyConfirmRequested,
    applyConfirmResolved,
    applyDelta,
    applyDone,
    applyToolFinished,
    applyToolStarted,
    beginTurn,
    dropAllTurns,
    getTurn,
    onStall,
    stallAfterMs,
    stalledConversationIds,
    subscribeTurn,
    sweepStalled,
} from "./turn.js";

const conversationId = "c1";

function delta(seq: number, text: string, messageId = "a1") {
    applyDelta({ conversationId, messageId, seq, text });
}

afterEach(() => {
    dropAllTurns();
});

describe("a turn assembles the streamed text", () => {
    test("deltas that arrive out of order are joined in sequence order", () => {
        beginTurn(conversationId, "u1");
        delta(2, " world");
        expect(getTurn(conversationId).text).toBe("");
        delta(1, "hello");
        expect(getTurn(conversationId).text).toBe("hello world");
        expect(getTurn(conversationId).chunks).toBe(2);
        delta(3, "!");
        expect(getTurn(conversationId).text).toBe("hello world!");
    });

    test("the first delta re-keys the turn from the asked message to the answer", () => {
        beginTurn(conversationId, "u1");
        delta(1, "first");
        expect(getTurn(conversationId).messageId).toBe("a1");
        delta(1, "again", "a2");
        expect(getTurn(conversationId).messageId).toBe("a2");
        expect(getTurn(conversationId).text).toBe("again");
        expect(getTurn(conversationId).chunks).toBe(1);
    });

    test("done replaces the text with the final answer and records the usage", () => {
        beginTurn(conversationId, "u1");
        delta(1, "partial");
        applyDone({ conversationId, messageId: "a1", text: "final", code: "", error: "", inputTokens: 12, outputTokens: 3, usd: 0.5 });
        const turn = getTurn(conversationId);
        expect(turn.status).toBe("done");
        expect(turn.text).toBe("final");
        expect(turn.usage).toStrictEqual({ inputTokens: 12, outputTokens: 3, usd: 0.5 });
        expect(turn.error).toBeNull();
    });

    test("done with an error is an error turn", () => {
        beginTurn(conversationId, "u1");
        applyDone({ conversationId, messageId: "a1", text: "", code: "EXTERNAL", error: "the model could not answer", inputTokens: 0, outputTokens: 0, usd: 0 });
        expect(getTurn(conversationId).status).toBe("error");
        expect(getTurn(conversationId).error).toBe("the model could not answer");
    });
});

describe("tool calls and confirmations ride on the turn", () => {
    test("a tool call runs and then settles with its outcome", () => {
        beginTurn(conversationId, "u1");
        applyToolStarted({ conversationId, callId: "k1", tool: "pages_list", args: { limit: 5 } });
        expect(getTurn(conversationId).tools).toStrictEqual([
            { callId: "k1", tool: "pages_list", args: { limit: 5 }, status: "running" },
        ]);
        applyToolFinished({ conversationId, callId: "k1", tool: "pages_list", result: { items: [] }, status: "ok", error: "", durationMs: 40 });
        expect(getTurn(conversationId).tools[0]).toStrictEqual({
            callId: "k1", tool: "pages_list", args: { limit: 5 }, status: "ok", result: { items: [] }, error: undefined, durationMs: 40,
        });
    });

    test("a requested confirmation pauses the turn until it is resolved", () => {
        beginTurn(conversationId, "u1");
        applyConfirmRequested({ conversationId, confirmationId: "p1", tool: "pages_delete", args: { id: "x" }, risk: "dangerous", summary: "delete" });
        expect(getTurn(conversationId).status).toBe("awaiting-confirm");
        expect(getTurn(conversationId).confirm?.confirmationId).toBe("p1");
        delta(1, "still");
        expect(getTurn(conversationId).status).toBe("awaiting-confirm");
        applyConfirmResolved({ conversationId, confirmationId: "p1", tool: "pages_delete", status: "rejected", result: null, error: "" });
        expect(getTurn(conversationId).status).toBe("streaming");
        expect(getTurn(conversationId).confirm).toBeNull();
    });
});

describe("a turn that stops reporting is stalled", () => {
    test("the sweep marks a silent turn stalled, keeps what it had, and tells the handlers", () => {
        const stalled: string[] = [];
        const stop = onStall((id) => {
            stalled.push(id);
        });
        beginTurn(conversationId, "u1");
        delta(1, "so far");
        const now = Date.now();
        expect(stalledConversationIds(now + stallAfterMs - 1)).toStrictEqual([]);
        sweepStalled(now + stallAfterMs);
        const turn = getTurn(conversationId);
        expect(turn.status).toBe("stalled");
        expect(turn.text).toBe("so far");
        expect(stalled).toStrictEqual([conversationId]);
        sweepStalled(now + stallAfterMs * 2);
        expect(stalled).toStrictEqual([conversationId]);
        stop();
    });

    test("a new turn clears the stall", () => {
        beginTurn(conversationId, "u1");
        sweepStalled(Date.now() + stallAfterMs);
        beginTurn(conversationId, "u2");
        expect(getTurn(conversationId).status).toBe("streaming");
        expect(getTurn(conversationId).text).toBe("");
    });

    test("a turn awaiting a confirmation is never stalled", () => {
        beginTurn(conversationId, "u1");
        applyConfirmRequested({ conversationId, confirmationId: "p1", tool: "pages_delete", args: {}, risk: "dangerous", summary: "delete" });
        expect(stalledConversationIds(Date.now() + stallAfterMs)).toStrictEqual([]);
    });
});

describe("subscribers and the lock", () => {
    test("a listener hears every published change and the lock drops every turn", () => {
        let heard = 0;
        const stop = subscribeTurn(conversationId, () => {
            heard += 1;
        });
        beginTurn(conversationId, "u1");
        delta(1, "a");
        expect(heard).toBe(2);
        dropAllTurns();
        expect(getTurn(conversationId).status).toBe("idle");
        expect(heard).toBe(3);
        stop();
    });
});
