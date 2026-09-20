import { afterEach, describe, expect, test } from "vitest";

import {
    applyConfirmRequested,
    applyConfirmResolved,
    applyDelta,
    applyDone,
    applyToolFinished,
    applyToolStarted,
    attachAssistant,
    beginStop,
    dropAllTurns,
    failTurn,
    getTurn,
    reconcile,
    settleUnreported,
    silenceAfterMs,
    silentConversationIds,
    startTurn,
    subscribeTurn,
} from "./turn.js";

const conversationId = "c1";

function delta(seq: number, text: string, messageId = "a1") {
    applyDelta({ conversationId, messageId, seq, text });
}

function done(code: string, error: string, text = "", messageId = "a1") {
    applyDone({ conversationId, messageId, text, code, error, inputTokens: 0, outputTokens: 0, usd: 0 });
}

function report(running: boolean, messageId = "", startedAt: string | null = null, lastSeq = 0) {
    return { running, messageId, startedAt, lastSeq };
}

afterEach(() => {
    dropAllTurns();
});

describe("a turn assembles the streamed text", () => {
    test("deltas that arrive out of order are joined in sequence order", () => {
        startTurn(conversationId);
        delta(2, " world");
        expect(getTurn(conversationId).text).toBe("");
        delta(1, "hello");
        expect(getTurn(conversationId).text).toBe("hello world");
        expect(getTurn(conversationId).chunks).toBe(2);
        delta(3, "!");
        expect(getTurn(conversationId).text).toBe("hello world!");
        expect(getTurn(conversationId).lastSeq).toBe(3);
    });

    test("a delta is applied by conversation even before Send has answered", () => {
        startTurn(conversationId);
        expect(getTurn(conversationId).assistantMessageId).toBeNull();
        delta(1, "early");
        expect(getTurn(conversationId).assistantMessageId).toBe("a1");
        expect(getTurn(conversationId).text).toBe("early");
    });

    test("a delta carrying another message starts a new turn instead of being dropped", () => {
        startTurn(conversationId);
        delta(1, "first");
        delta(1, "again", "a2");
        const turn = getTurn(conversationId);
        expect(turn.assistantMessageId).toBe("a2");
        expect(turn.text).toBe("again");
        expect(turn.chunks).toBe(1);
        expect(turn.status).toBe("working");
    });

    test("done replaces the text with the final answer and records the usage", () => {
        startTurn(conversationId);
        delta(1, "partial");
        applyDone({
            conversationId,
            messageId: "a1",
            text: "final",
            code: "",
            error: "",
            inputTokens: 12,
            outputTokens: 3,
            usd: 0.5,
        });
        const turn = getTurn(conversationId);
        expect(turn.status).toBe("done");
        expect(turn.end).toBe("answered");
        expect(turn.text).toBe("final");
        expect(turn.usage).toStrictEqual({ inputTokens: 12, outputTokens: 3, usd: 0.5 });
    });
});

describe("a turn settles on its terminal event", () => {
    test("a coded failure is an error turn carrying the code and the message", () => {
        startTurn(conversationId);
        done("EXTERNAL", "the model could not answer");
        const turn = getTurn(conversationId);
        expect(turn.status).toBe("error");
        expect(turn.end).toBe("failed");
        expect(turn.code).toBe("EXTERNAL");
        expect(turn.message).toBe("the model could not answer");
    });

    test("a cancelled turn is done with a stopped marker and the described reason", () => {
        startTurn(conversationId);
        beginStop(conversationId);
        expect(getTurn(conversationId).status).toBe("stopping");
        done("CANCELLED", "you stopped the turn");
        const turn = getTurn(conversationId);
        expect(turn.status).toBe("done");
        expect(turn.end).toBe("stopped");
        expect(turn.message).toBe("you stopped the turn");
    });

    test("done that lands before Send resolves is not overwritten by the answer", () => {
        const turnSeq = startTurn(conversationId);
        done("UNAUTHORIZED", "no key is stored for this provider");
        attachAssistant(conversationId, "a1", turnSeq);
        const turn = getTurn(conversationId);
        expect(turn.status).toBe("error");
        expect(turn.code).toBe("UNAUTHORIZED");
    });

    test("a rejected Send fails the turn it started and nothing else", () => {
        const turnSeq = startTurn(conversationId);
        failTurn(conversationId, "LOCKED", "Postulator is locked", turnSeq);
        expect(getTurn(conversationId).status).toBe("error");
        const later = startTurn(conversationId);
        failTurn(conversationId, "LOCKED", "Postulator is locked", later - 1);
        expect(getTurn(conversationId).status).toBe("working");
    });
});

describe("a confirmation parks the turn", () => {
    test("a requested confirmation moves the turn to awaiting-confirm", () => {
        startTurn(conversationId);
        applyConfirmRequested({
            conversationId,
            confirmationId: "p1",
            tool: "pages_update",
            args: {},
            risk: "write",
            summary: "update a page",
        });
        expect(getTurn(conversationId).status).toBe("awaiting-confirm");
    });

    test("resolving a confirmation does not resume the turn on its own", () => {
        startTurn(conversationId);
        applyConfirmRequested({
            conversationId,
            confirmationId: "p1",
            tool: "pages_update",
            args: {},
            risk: "write",
            summary: "update a page",
        });
        applyConfirmResolved({
            conversationId,
            confirmationId: "p1",
            tool: "pages_update",
            status: "executed",
            result: null,
            error: "",
        });
        expect(getTurn(conversationId).status).toBe("awaiting-confirm");
        expect(getTurn(conversationId).confirm).toBeNull();
        expect(reconcile(conversationId, report(false))).toBe("refetch");
        settleUnreported(conversationId, true);
        expect(getTurn(conversationId).status).toBe("idle");
    });

    test("a confirmation resolved while Go is still running resumes the turn", () => {
        startTurn(conversationId);
        applyConfirmRequested({
            conversationId,
            confirmationId: "p1",
            tool: "pages_update",
            args: {},
            risk: "write",
            summary: "update a page",
        });
        applyConfirmResolved({
            conversationId,
            confirmationId: "p1",
            tool: "pages_update",
            status: "executed",
            result: null,
            error: "",
        });
        expect(reconcile(conversationId, report(true, "a1"))).toBe("none");
        expect(getTurn(conversationId).status).toBe("working");
    });
});

describe("reconciling against the turn registry", () => {
    test("a silent turn that Go is not running asks for a refetch and settles", () => {
        startTurn(conversationId, Date.now() - silenceAfterMs - 1);
        expect(silentConversationIds(Date.now())).toStrictEqual([conversationId]);
        expect(reconcile(conversationId, report(false))).toBe("refetch");
        settleUnreported(conversationId, false);
        const turn = getTurn(conversationId);
        expect(turn.status).toBe("error");
        expect(turn.end).toBe("lost");
    });

    test("an answer that arrived while the store waited drops the live turn", () => {
        startTurn(conversationId);
        expect(reconcile(conversationId, report(false))).toBe("refetch");
        settleUnreported(conversationId, true);
        expect(getTurn(conversationId).status).toBe("idle");
    });

    test("a terminal event between the status read and the settle wins", () => {
        startTurn(conversationId);
        expect(reconcile(conversationId, report(false))).toBe("refetch");
        done("", "", "the answer");
        settleUnreported(conversationId, false);
        expect(getTurn(conversationId).status).toBe("done");
        expect(getTurn(conversationId).text).toBe("the answer");
    });

    test("a window that opened mid-turn adopts the running turn", () => {
        const startedAt = new Date(Date.now() - 5000).toISOString();
        expect(reconcile(conversationId, report(true, "a9", startedAt, 7))).toBe("adopt");
        const turn = getTurn(conversationId);
        expect(turn.status).toBe("working");
        expect(turn.assistantMessageId).toBe("a9");
        expect(turn.lastSeq).toBe(7);
        expect(turn.startedAt).toBe(Date.parse(startedAt));
    });

    test("a turn restarted by a late delta is still swept by the silence timer", () => {
        startTurn(conversationId);
        const stale = Date.now() - silenceAfterMs - 1;
        applyDelta({ conversationId, messageId: "a2", seq: 1, text: "late" }, stale);
        expect(silentConversationIds(Date.now())).toStrictEqual([conversationId]);
        expect(reconcile(conversationId, report(false))).toBe("refetch");
        settleUnreported(conversationId, false);
        expect(getTurn(conversationId).end).toBe("lost");
    });
});

describe("tool calls and the lock", () => {
    test("tool events are applied by conversation and finish in place", () => {
        startTurn(conversationId);
        applyToolStarted({ conversationId, callId: "t1", tool: "pages_list", args: {} });
        applyToolFinished({
            conversationId,
            callId: "t1",
            tool: "pages_list",
            result: { items: [] },
            status: "ok",
            error: "",
            durationMs: 12,
        });
        const tools = getTurn(conversationId).tools;
        expect(tools).toHaveLength(1);
        expect(tools[0]?.status).toBe("ok");
        expect(tools[0]?.durationMs).toBe(12);
    });

    test("the lock resets every turn and keeps its subscribers", () => {
        let notified = 0;
        const stop = subscribeTurn(conversationId, () => {
            notified += 1;
        });
        startTurn(conversationId);
        expect(notified).toBe(1);
        dropAllTurns();
        expect(notified).toBe(2);
        expect(getTurn(conversationId).status).toBe("idle");
        startTurn(conversationId);
        expect(notified).toBe(3);
        stop();
    });
});
