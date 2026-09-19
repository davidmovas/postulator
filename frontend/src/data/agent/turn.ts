import type {
    AgentConfirmRequestedPayload,
    AgentConfirmResolvedPayload,
    AgentDeltaPayload,
    AgentDonePayload,
    AgentToolFinishedPayload,
    AgentToolStartedPayload,
} from "../../generated/events.js";

export type TurnStatus = "idle" | "streaming" | "awaiting-confirm" | "done" | "error";

export type ToolCallStatus = "running" | "ok" | "error";

export interface AgentToolCall {
    callId: string;
    tool: string;
    args: unknown;
    status: ToolCallStatus;
    result?: unknown;
    error?: string;
    durationMs?: number;
}

export interface AgentConfirmation {
    confirmationId: string;
    tool: string;
    args: unknown;
    risk: string;
    summary: string;
}

export interface TurnUsage {
    inputTokens: number;
    outputTokens: number;
    usd: number;
}

export interface Turn {
    messageId: string | null;
    text: string;
    chunks: number;
    tools: readonly AgentToolCall[];
    confirm: AgentConfirmation | null;
    status: TurnStatus;
    error: string | null;
    usage: TurnUsage | null;
}

interface TurnLog {
    conversationId: string;
    turn: Turn;
    snapshot: Turn;
    buffered: Map<number, string>;
    nextSeq: number;
    startedAt: number;
    listeners: Set<() => void>;
}

export const stallAfterMs = 90_000;

const turns = new Map<string, TurnLog>();

const idleTurn: Turn = Object.freeze({
    messageId: null,
    text: "",
    chunks: 0,
    tools: [] as readonly AgentToolCall[],
    confirm: null,
    status: "idle" as TurnStatus,
    error: null,
    usage: null,
});

type StallHandler = (conversationId: string) => void;

const stallHandlers = new Set<StallHandler>();

function snapshotOf(held: TurnLog): Turn {
    return Object.freeze({
        messageId: held.turn.messageId,
        text: held.turn.text,
        chunks: held.turn.chunks,
        tools: held.turn.tools.slice(),
        confirm: held.turn.confirm,
        status: held.turn.status,
        error: held.turn.error,
        usage: held.turn.usage,
    });
}

function reach(conversationId: string): TurnLog {
    const held = turns.get(conversationId);
    if (held !== undefined) {
        return held;
    }
    const created: TurnLog = {
        conversationId,
        turn: {
            messageId: null,
            text: "",
            chunks: 0,
            tools: [],
            confirm: null,
            status: "idle",
            error: null,
            usage: null,
        },
        snapshot: idleTurn,
        buffered: new Map<number, string>(),
        nextSeq: 1,
        startedAt: 0,
        listeners: new Set<() => void>(),
    };
    turns.set(conversationId, created);
    return created;
}

function publish(held: TurnLog): void {
    held.snapshot = snapshotOf(held);
    held.listeners.forEach((listener) => {
        listener();
    });
}

export function beginTurn(conversationId: string, messageId: string): void {
    const held = reach(conversationId);
    held.turn = {
        messageId,
        text: "",
        chunks: 0,
        tools: [],
        confirm: null,
        status: "streaming",
        error: null,
        usage: null,
    };
    held.buffered.clear();
    held.nextSeq = 1;
    held.startedAt = Date.now();
    publish(held);
}

export function applyDelta(payload: AgentDeltaPayload): void {
    const held = reach(payload.conversationId);
    if (held.turn.messageId !== payload.messageId) {
        held.turn.messageId = payload.messageId;
        held.turn.text = "";
        held.turn.chunks = 0;
        held.buffered.clear();
        held.nextSeq = 1;
    }
    held.turn.status = held.turn.status === "awaiting-confirm" ? "awaiting-confirm" : "streaming";
    held.buffered.set(payload.seq, payload.text);
    let advanced = false;
    for (;;) {
        const next = held.buffered.get(held.nextSeq);
        if (next === undefined) {
            break;
        }
        held.turn.text += next;
        held.turn.chunks += 1;
        held.buffered.delete(held.nextSeq);
        held.nextSeq += 1;
        advanced = true;
    }
    if (advanced) {
        publish(held);
    }
}

export function applyToolStarted(payload: AgentToolStartedPayload): void {
    const held = reach(payload.conversationId);
    held.turn.tools = [
        ...held.turn.tools,
        { callId: payload.callId, tool: payload.tool, args: payload.args, status: "running" },
    ];
    publish(held);
}

export function applyToolFinished(payload: AgentToolFinishedPayload): void {
    const held = reach(payload.conversationId);
    held.turn.tools = held.turn.tools.map((call) =>
        call.callId === payload.callId
            ? {
                  ...call,
                  status: payload.status === "ok" ? "ok" : "error",
                  result: payload.result,
                  error: payload.error === "" ? undefined : payload.error,
                  durationMs: payload.durationMs,
              }
            : call,
    );
    publish(held);
}

export function applyConfirmRequested(payload: AgentConfirmRequestedPayload): void {
    const held = reach(payload.conversationId);
    held.turn.confirm = {
        confirmationId: payload.confirmationId,
        tool: payload.tool,
        args: payload.args,
        risk: payload.risk,
        summary: payload.summary,
    };
    held.turn.status = "awaiting-confirm";
    publish(held);
}

export function applyConfirmResolved(payload: AgentConfirmResolvedPayload): void {
    const held = reach(payload.conversationId);
    if (held.turn.confirm?.confirmationId === payload.confirmationId) {
        held.turn.confirm = null;
    }
    if (held.turn.status === "awaiting-confirm") {
        held.turn.status = "streaming";
    }
    publish(held);
}

export function applyDone(payload: AgentDonePayload): void {
    const held = reach(payload.conversationId);
    held.turn.messageId = payload.messageId;
    held.turn.text = payload.text;
    held.turn.confirm = null;
    held.turn.error = payload.error === "" ? null : payload.error;
    held.turn.status = payload.error === "" ? "done" : "error";
    held.turn.usage = {
        inputTokens: payload.inputTokens,
        outputTokens: payload.outputTokens,
        usd: payload.usd,
    };
    held.buffered.clear();
    held.startedAt = 0;
    publish(held);
}

export function resetTurn(conversationId: string): void {
    const held = turns.get(conversationId);
    if (held === undefined) {
        return;
    }
    held.turn = {
        messageId: held.turn.messageId,
        text: held.turn.text,
        chunks: held.turn.chunks,
        tools: held.turn.tools,
        confirm: null,
        status: "idle",
        error: null,
        usage: held.turn.usage,
    };
    held.buffered.clear();
    held.startedAt = 0;
    publish(held);
}

export function stalledConversationIds(now: number): string[] {
    const out: string[] = [];
    turns.forEach((held) => {
        if (held.turn.status === "streaming" && held.startedAt > 0 && now - held.startedAt >= stallAfterMs) {
            out.push(held.conversationId);
        }
    });
    return out;
}

export function sweepStalled(now: number): void {
    for (const conversationId of stalledConversationIds(now)) {
        resetTurn(conversationId);
        stallHandlers.forEach((handler) => {
            handler(conversationId);
        });
    }
}

export function onStall(handler: StallHandler): () => void {
    stallHandlers.add(handler);
    return () => {
        stallHandlers.delete(handler);
    };
}

export function subscribeTurn(conversationId: string, listener: () => void): () => void {
    const held = reach(conversationId);
    held.listeners.add(listener);
    return () => {
        held.listeners.delete(listener);
    };
}

export function getTurn(conversationId: string): Turn {
    const held = turns.get(conversationId);
    return held === undefined ? idleTurn : held.snapshot;
}

export function dropAllTurns(): void {
    turns.forEach((held) => {
        held.snapshot = idleTurn;
        held.listeners.forEach((listener) => {
            listener();
        });
        held.listeners.clear();
    });
    turns.clear();
}
