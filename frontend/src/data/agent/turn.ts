import type {
    AgentConfirmRequestedPayload,
    AgentConfirmResolvedPayload,
    AgentDeltaPayload,
    AgentDonePayload,
    AgentToolFinishedPayload,
    AgentToolStartedPayload,
} from "../../generated/events.js";
import type {
    AgentToolCall,
    ReconcileAction,
    Turn,
    TurnEnd,
    TurnReport,
    TurnStatus,
} from "./model.js";
import { cancelledCode, isActive, silenceAfterMs, toolCallStatus } from "./model.js";

export type {
    AgentConfirmation,
    AgentToolCall,
    ReconcileAction,
    ToolCallStatus,
    Turn,
    TurnEnd,
    TurnReport,
    TurnStatus,
    TurnUsage,
} from "./model.js";
export { budgetCode, cancelledCode, isActive, silenceAfterMs, toolCallStatus } from "./model.js";

interface TurnLog {
    conversationId: string;
    turn: Turn;
    snapshot: Turn;
    buffered: Map<number, string>;
    nextSeq: number;
    pendingReconcile: boolean;
    listeners: Set<() => void>;
}

const idleTurn: Turn = Object.freeze({
    status: "idle" as TurnStatus,
    end: null,
    assistantMessageId: null,
    startedAt: 0,
    lastEventAt: 0,
    lastSeq: 0,
    text: "",
    chunks: 0,
    tools: Object.freeze([]) as readonly AgentToolCall[],
    confirm: null,
    code: "",
    message: "",
    usage: null,
    turnSeq: 0,
});

const turns = new Map<string, TurnLog>();

let issued = 0;

function blank(): Turn {
    return { ...idleTurn, tools: [] };
}

function snapshotOf(held: TurnLog): Turn {
    return Object.freeze({ ...held.turn, tools: held.turn.tools.slice() });
}

function reach(conversationId: string): TurnLog {
    const held = turns.get(conversationId);
    if (held !== undefined) {
        return held;
    }
    const created: TurnLog = {
        conversationId,
        turn: blank(),
        snapshot: idleTurn,
        buffered: new Map<number, string>(),
        nextSeq: 1,
        pendingReconcile: false,
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

function restart(held: TurnLog, assistantMessageId: string | null, at: number): void {
    issued += 1;
    held.turn = {
        ...blank(),
        status: "working",
        assistantMessageId,
        startedAt: at,
        lastEventAt: at,
        turnSeq: issued,
    };
    held.buffered.clear();
    held.nextSeq = 1;
    held.pendingReconcile = false;
}

export function startTurn(conversationId: string, at: number = Date.now()): number {
    const held = reach(conversationId);
    restart(held, null, at);
    publish(held);
    return held.turn.turnSeq;
}

export function attachAssistant(conversationId: string, assistantMessageId: string, turnSeq: number): void {
    const held = turns.get(conversationId);
    if (held === undefined || held.turn.turnSeq !== turnSeq || !isActive(held.turn.status)) {
        return;
    }
    if (held.turn.assistantMessageId === assistantMessageId) {
        return;
    }
    held.turn.assistantMessageId = assistantMessageId;
    publish(held);
}

export function failTurn(conversationId: string, code: string, message: string, turnSeq: number): void {
    const held = turns.get(conversationId);
    if (held === undefined || held.turn.turnSeq !== turnSeq || !isActive(held.turn.status)) {
        return;
    }
    settle(held, "error", "failed", code, message);
}

function settle(held: TurnLog, status: TurnStatus, end: TurnEnd, code: string, message: string): void {
    held.turn.status = status;
    held.turn.end = end;
    held.turn.code = code;
    held.turn.message = message;
    held.turn.confirm = null;
    held.turn.lastEventAt = Date.now();
    held.buffered.clear();
    held.pendingReconcile = false;
    publish(held);
}

export function beginStop(conversationId: string): void {
    const held = turns.get(conversationId);
    if (held === undefined || (held.turn.status !== "working" && held.turn.status !== "awaiting-confirm")) {
        return;
    }
    held.turn.status = "stopping";
    held.turn.lastEventAt = Date.now();
    publish(held);
}

export function applyDelta(payload: AgentDeltaPayload, at: number = Date.now()): void {
    const held = reach(payload.conversationId);
    const known = held.turn.assistantMessageId;
    if (known !== null && known !== payload.messageId) {
        restart(held, payload.messageId, at);
    } else if (!isActive(held.turn.status)) {
        restart(held, payload.messageId, at);
    } else if (known === null) {
        held.turn.assistantMessageId = payload.messageId;
    }
    if (held.turn.status !== "awaiting-confirm") {
        held.turn.status = "working";
    }
    held.turn.lastEventAt = at;
    held.turn.lastSeq = Math.max(held.turn.lastSeq, payload.seq);
    held.pendingReconcile = false;
    held.buffered.set(payload.seq, payload.text);
    for (;;) {
        const next = held.buffered.get(held.nextSeq);
        if (next === undefined) {
            break;
        }
        held.turn.text += next;
        held.turn.chunks += 1;
        held.buffered.delete(held.nextSeq);
        held.nextSeq += 1;
    }
    publish(held);
}

function touch(held: TurnLog, at: number): void {
    held.turn.lastEventAt = at;
    held.pendingReconcile = false;
    if (!isActive(held.turn.status)) {
        held.turn.status = "working";
        held.turn.end = null;
        if (held.turn.startedAt === 0) {
            held.turn.startedAt = at;
        }
    }
}

export function applyToolStarted(payload: AgentToolStartedPayload, at: number = Date.now()): void {
    const held = reach(payload.conversationId);
    touch(held, at);
    held.turn.tools = [
        ...held.turn.tools,
        { callId: payload.callId, tool: payload.tool, args: payload.args, status: "running" },
    ];
    publish(held);
}

export function applyToolFinished(payload: AgentToolFinishedPayload, at: number = Date.now()): void {
    const held = reach(payload.conversationId);
    touch(held, at);
    held.turn.tools = held.turn.tools.map((call) =>
        call.callId === payload.callId
            ? {
                  ...call,
                  status: toolCallStatus(payload.status),
                  result: payload.result,
                  error: payload.error === "" ? undefined : payload.error,
                  durationMs: payload.durationMs,
              }
            : call,
    );
    publish(held);
}

export function applyConfirmRequested(payload: AgentConfirmRequestedPayload, at: number = Date.now()): void {
    const held = reach(payload.conversationId);
    touch(held, at);
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

export function applyConfirmResolved(payload: AgentConfirmResolvedPayload, at: number = Date.now()): void {
    const held = reach(payload.conversationId);
    if (held.turn.confirm?.confirmationId === payload.confirmationId) {
        held.turn.confirm = null;
    }
    held.turn.lastEventAt = at;
    publish(held);
}

export function applyDone(payload: AgentDonePayload, at: number = Date.now()): void {
    const held = reach(payload.conversationId);
    if (payload.messageId !== "") {
        held.turn.assistantMessageId = payload.messageId;
    }
    if (payload.text !== "") {
        held.turn.text = payload.text;
    }
    held.turn.usage = {
        inputTokens: payload.inputTokens,
        outputTokens: payload.outputTokens,
        usd: payload.usd,
    };
    held.turn.lastEventAt = at;
    if (payload.code === "") {
        settle(held, "done", "answered", "", "");
        return;
    }
    if (payload.code === cancelledCode) {
        settle(held, "done", "stopped", payload.code, payload.error);
        return;
    }
    settle(held, "error", "failed", payload.code, payload.error);
}

function adoptedStart(startedAt: string | null, now: number): number {
    if (startedAt === null) {
        return now;
    }
    const parsed = Date.parse(startedAt);
    return Number.isNaN(parsed) ? now : parsed;
}

export function reconcile(conversationId: string, report: TurnReport, now: number = Date.now()): ReconcileAction {
    const held = reach(conversationId);
    if (isActive(held.turn.status) && !report.running) {
        held.pendingReconcile = true;
        return "refetch";
    }
    if (!isActive(held.turn.status) && report.running) {
        restart(held, report.messageId === "" ? null : report.messageId, adoptedStart(report.startedAt, now));
        held.turn.lastSeq = report.lastSeq;
        publish(held);
        return "adopt";
    }
    if (held.turn.status === "awaiting-confirm" && report.running && held.turn.confirm === null) {
        held.turn.status = "working";
        held.turn.lastEventAt = now;
        publish(held);
    }
    return "none";
}

export function settleUnreported(conversationId: string, answered: boolean): void {
    const held = turns.get(conversationId);
    if (held === undefined || !held.pendingReconcile || !isActive(held.turn.status)) {
        return;
    }
    if (answered) {
        held.turn = blank();
        held.buffered.clear();
        held.nextSeq = 1;
        held.pendingReconcile = false;
        publish(held);
        return;
    }
    settle(held, "error", "lost", "", "");
}

export function silentConversationIds(now: number, afterMs: number = silenceAfterMs): string[] {
    const out: string[] = [];
    turns.forEach((held) => {
        if (isActive(held.turn.status) && held.turn.lastEventAt > 0 && now - held.turn.lastEventAt >= afterMs) {
            out.push(held.conversationId);
        }
    });
    return out;
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
        held.turn = blank();
        held.buffered.clear();
        held.nextSeq = 1;
        held.pendingReconcile = false;
        publish(held);
    });
}
