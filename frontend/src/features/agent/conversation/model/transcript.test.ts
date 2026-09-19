import { describe, expect, it } from "vitest";

import type { Turn } from "../../../../data/agent/turn.js";
import type { Message, PendingAction } from "../../../../data/types.js";
import { cancelledTurnError, rows } from "./transcript.js";

const idle: Turn = {
    messageId: null,
    text: "",
    chunks: 0,
    tools: [],
    confirm: null,
    status: "idle",
    error: null,
    usage: null,
};

function message(seq: number, role: string, text: string, extra: Partial<Message> = {}): Message {
    return {
        id: `m${seq}`,
        conversationId: "c1",
        seq,
        role,
        text,
        createdAt: "2026-09-19T10:00:00Z",
        ...extra,
    };
}

function action(id: string, status = "pending"): PendingAction {
    return {
        id,
        conversationId: "c1",
        tool: "pages_delete",
        args: { id: "p1" },
        summary: "delete a page",
        status,
        createdAt: "2026-09-19T10:00:00Z",
        updatedAt: "2026-09-19T10:00:00Z",
    };
}

describe("rows over the saved transcript", () => {
    it("turns saved messages into user, assistant and tool rows in sequence order", () => {
        const saved = [
            message(1, "user", "hello"),
            message(2, "tool", "", { tool: "pages_list", callId: "k1", payload: { items: [] } }),
            message(3, "assistant", "nothing there"),
        ];
        expect(rows(saved, idle, []).map((row) => row.kind)).toStrictEqual(["user", "tool", "assistant"]);
        const tool = rows(saved, idle, [])[1];
        expect(tool).toMatchObject({ kind: "tool", callId: "k1", tool: "pages_list", status: "ok", live: false });
    });

    it("reads a saved tool failure off its text", () => {
        const saved = [message(1, "tool", "no such page", { tool: "pages_get", callId: "k1", payload: null })];
        expect(rows(saved, idle, [])[0]).toMatchObject({ kind: "tool", status: "error", error: "no such page" });
    });

    it("lists every pending action as a card even when no turn is live", () => {
        const saved = [message(1, "user", "delete it")];
        const listed = rows(saved, idle, [action("a1"), action("a2", "executed")]);
        expect(listed.map((row) => row.kind)).toStrictEqual(["user", "confirm"]);
        expect(listed[1]).toMatchObject({ kind: "confirm", id: "a1" });
    });
});

describe("rows over a live turn", () => {
    it("shows a working row while the answer has not started", () => {
        const turn: Turn = { ...idle, messageId: "u1", status: "streaming" };
        expect(rows([message(1, "user", "hi")], turn, []).map((row) => row.kind)).toStrictEqual(["user", "working"]);
    });

    it("streams the live text under the saved rows", () => {
        const turn: Turn = { ...idle, messageId: "a1", text: "so far", chunks: 2, status: "streaming" };
        const listed = rows([message(1, "user", "hi")], turn, []);
        expect(listed.map((row) => row.kind)).toStrictEqual(["user", "streaming"]);
        expect(listed[1]).toMatchObject({ kind: "streaming", text: "so far" });
    });

    it("hides a live tool call once its saved row has arrived", () => {
        const turn: Turn = {
            ...idle,
            messageId: "a1",
            status: "streaming",
            tools: [
                { callId: "k1", tool: "pages_list", args: {}, status: "ok", result: {}, durationMs: 5 },
                { callId: "k2", tool: "pages_get", args: {}, status: "running" },
            ],
        };
        const saved = [message(1, "user", "hi"), message(2, "tool", "", { tool: "pages_list", callId: "k1", payload: {} })];
        const listed = rows(saved, turn, []);
        expect(listed.map((row) => row.kind)).toStrictEqual(["user", "tool", "tool", "working"]);
        expect(listed[1]).toMatchObject({ callId: "k1", live: false });
        expect(listed[2]).toMatchObject({ callId: "k2", live: true, status: "running" });
    });

    it("hides the live answer once the saved assistant row carries the same id", () => {
        const turn: Turn = {
            ...idle,
            messageId: "a1",
            text: "final",
            status: "done",
            usage: { inputTokens: 10, outputTokens: 2, usd: 0.01 },
        };
        const saved = [message(1, "user", "hi"), message(2, "assistant", "final", { id: "a1" })];
        const listed = rows(saved, turn, []);
        expect(listed.map((row) => row.kind)).toStrictEqual(["user", "assistant"]);
        expect(listed[1]).toMatchObject({ id: "a1", usage: { inputTokens: 10, outputTokens: 2, usd: 0.01 } });
    });

    it("shows the finished answer live until the saved row lands", () => {
        const turn: Turn = { ...idle, messageId: "a1", text: "final", status: "done" };
        const listed = rows([message(1, "user", "hi")], turn, []);
        expect(listed[1]).toMatchObject({ kind: "assistant", id: "a1", text: "final", live: true });
    });

    it("renders a live confirmation from the turn until the pending list has it", () => {
        const turn: Turn = {
            ...idle,
            messageId: "a1",
            status: "awaiting-confirm",
            confirm: { confirmationId: "a1", tool: "pages_delete", args: {}, risk: "dangerous", summary: "delete" },
        };
        const fromTurn = rows([], turn, []);
        expect(fromTurn.map((row) => row.kind)).toStrictEqual(["confirm"]);
        expect(fromTurn[0]).toMatchObject({ id: "a1", action: null });
        const fromList = rows([], turn, [action("a1")]);
        expect(fromList).toHaveLength(1);
        expect(fromList[0]).toMatchObject({ id: "a1", action: { id: "a1" } });
    });

    it("tells a stop from a failure and reports a stall", () => {
        const stopped: Turn = { ...idle, status: "error", error: cancelledTurnError };
        expect(rows([], stopped, []).map((row) => row.kind)).toStrictEqual(["cancelled"]);
        const failed: Turn = { ...idle, status: "error", error: "the model could not answer" };
        expect(rows([], failed, [])[0]).toMatchObject({ kind: "error", message: "the model could not answer" });
        const stalled: Turn = { ...idle, status: "stalled" };
        expect(rows([], stalled, []).map((row) => row.kind)).toStrictEqual(["stalled"]);
    });
});
