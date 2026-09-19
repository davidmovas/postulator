import type { AgentConfirmation, Turn, TurnUsage } from "../../../../data/agent/turn.js";
import type { Message, PendingAction } from "../../../../data/types.js";
import type { Timestamp } from "../../../../data/wire.js";

export const cancelledTurnError = "the agent turn was stopped";

export type ToolRowStatus = "running" | "ok" | "error";

export type Row =
    | { kind: "user"; id: string; text: string; at: Timestamp }
    | { kind: "assistant"; id: string; text: string; at: Timestamp; usage: TurnUsage | null; live: boolean }
    | {
          kind: "tool";
          id: string;
          callId: string;
          tool: string;
          status: ToolRowStatus;
          durationMs: number | null;
          result: unknown;
          error: string | null;
          live: boolean;
      }
    | { kind: "confirm"; id: string; action: PendingAction | null; live: AgentConfirmation | null }
    | { kind: "streaming"; id: string; text: string }
    | { kind: "working"; id: string }
    | { kind: "cancelled"; id: string }
    | { kind: "error"; id: string; message: string }
    | { kind: "stalled"; id: string };

function savedRow(message: Message, turn: Turn): Row | null {
    switch (message.role) {
        case "user":
            return { kind: "user", id: message.id, text: message.text, at: message.createdAt };
        case "assistant":
            return {
                kind: "assistant",
                id: message.id,
                text: message.text,
                at: message.createdAt,
                usage: turn.messageId === message.id ? turn.usage : null,
                live: false,
            };
        case "tool":
            return {
                kind: "tool",
                id: message.id,
                callId: message.callId ?? "",
                tool: message.tool ?? "",
                status: message.text === "" ? "ok" : "error",
                durationMs: null,
                result: message.payload ?? null,
                error: message.text === "" ? null : message.text,
                live: false,
            };
        default:
            return null;
    }
}

function liveRows(turn: Turn, saved: readonly Message[], pending: readonly PendingAction[]): Row[] {
    const out: Row[] = [];
    const savedCalls = new Set(saved.filter((held) => held.role === "tool").map((held) => held.callId ?? ""));
    const active = turn.status !== "idle";

    if (active) {
        for (const call of turn.tools) {
            if (savedCalls.has(call.callId)) {
                continue;
            }
            out.push({
                kind: "tool",
                id: `live:${call.callId}`,
                callId: call.callId,
                tool: call.tool,
                status: call.status,
                durationMs: call.durationMs ?? null,
                result: call.result ?? null,
                error: call.error ?? null,
                live: true,
            });
        }
    }

    const listed = new Set<string>();
    for (const held of pending) {
        if (held.status !== "pending") {
            continue;
        }
        listed.add(held.id);
        out.push({ kind: "confirm", id: held.id, action: held, live: null });
    }
    if (active && turn.confirm !== null && !listed.has(turn.confirm.confirmationId)) {
        out.push({ kind: "confirm", id: turn.confirm.confirmationId, action: null, live: turn.confirm });
    }

    if (!active) {
        return out;
    }

    const answered = turn.messageId !== null && saved.some((held) => held.id === turn.messageId);
    switch (turn.status) {
        case "streaming":
        case "awaiting-confirm":
            if (turn.text !== "" && !answered) {
                out.push({ kind: "streaming", id: "live:text", text: turn.text });
            } else if (turn.status === "streaming") {
                out.push({ kind: "working", id: "live:working" });
            }
            break;
        case "done":
            if (turn.text !== "" && !answered && turn.messageId !== null) {
                out.push({
                    kind: "assistant",
                    id: turn.messageId,
                    text: turn.text,
                    at: null,
                    usage: turn.usage,
                    live: true,
                });
            }
            break;
        case "error":
            if (turn.error === cancelledTurnError) {
                out.push({ kind: "cancelled", id: "live:cancelled" });
            } else {
                out.push({ kind: "error", id: "live:error", message: turn.error ?? "" });
            }
            break;
        case "stalled":
            out.push({ kind: "stalled", id: "live:stalled" });
            break;
        default:
            break;
    }
    return out;
}

export function rows(saved: readonly Message[], turn: Turn, pending: readonly PendingAction[]): Row[] {
    const out: Row[] = [];
    for (const message of saved) {
        const row = savedRow(message, turn);
        if (row !== null) {
            out.push(row);
        }
    }
    return [...out, ...liveRows(turn, saved, pending)];
}
