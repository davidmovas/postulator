import type { AgentConfirmation, ToolCallStatus, Turn, TurnUsage } from "../../../../data/agent/turn.js";
import { isActive, toolCallStatus } from "../../../../data/agent/turn.js";
import type { Message, PendingAction, Plan } from "../../../../data/types.js";
import type { Timestamp } from "../../../../data/wire.js";
import { planIdOf, planTool, proposedPlanOf } from "../../plans/model.js";
import { refusedText, truncationOf } from "./tools.js";

export type ToolRowStatus = "running" | "ok" | "cut" | "denied" | "error";

export type Row =
    | { kind: "user"; id: string; text: string; at: Timestamp }
    | { kind: "assistant"; id: string; text: string; at: Timestamp; usage: TurnUsage | null; live: boolean }
    | {
          kind: "tool";
          id: string;
          callId: string;
          tool: string;
          args: unknown;
          status: ToolRowStatus;
          durationMs: number | null;
          result: unknown;
          error: string | null;
          live: boolean;
      }
    | { kind: "confirm"; id: string; action: PendingAction | null; live: AgentConfirmation | null }
    | { kind: "plan"; id: string; plan: Plan }
    | { kind: "streaming"; id: string; text: string }
    | { kind: "working"; id: string }
    | { kind: "stopped"; id: string; detail: string }
    | { kind: "failed"; id: string; code: string; message: string }
    | { kind: "lost"; id: string };

function settledStatus(reported: ToolCallStatus, result: unknown): ToolRowStatus {
    if (reported === "ok" && truncationOf(result) !== null) {
        return "cut";
    }
    return reported;
}

export function savedStatus(message: Message): ToolCallStatus {
    const recorded = message.toolStatus ?? "";
    if (recorded !== "") {
        return toolCallStatus(recorded);
    }
    if (message.text === "") {
        return "ok";
    }
    return refusedText(message.text) ? "denied" : "error";
}

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
                usage: turn.assistantMessageId === message.id ? turn.usage : null,
                live: false,
            };
        case "tool":
            return {
                kind: "tool",
                id: message.id,
                callId: message.callId ?? "",
                tool: message.tool ?? "",
                args: null,
                status: settledStatus(savedStatus(message), message.payload ?? null),
                durationMs: null,
                result: message.payload ?? null,
                error: message.text === "" ? null : message.text,
                live: false,
            };
        default:
            return null;
    }
}

function partial(turn: Turn, answered: boolean): Row | null {
    if (turn.text === "" || answered) {
        return null;
    }
    return {
        kind: "assistant",
        id: turn.assistantMessageId ?? "live:text",
        text: turn.text,
        at: null,
        usage: turn.usage,
        live: true,
    };
}

function liveRows(turn: Turn, saved: readonly Message[]): Row[] {
    const out: Row[] = [];
    const savedCalls = new Set(saved.filter((held) => held.role === "tool").map((held) => held.callId ?? ""));
    const running = isActive(turn.status);
    const answered =
        turn.assistantMessageId !== null && saved.some((held) => held.id === turn.assistantMessageId);

    if (running || turn.status === "done") {
        for (const call of turn.tools) {
            if (savedCalls.has(call.callId)) {
                continue;
            }
            out.push({
                kind: "tool",
                id: `live:${call.callId}`,
                callId: call.callId,
                tool: call.tool,
                args: call.args ?? null,
                status: call.status === "running" ? "running" : settledStatus(call.status, call.result ?? null),
                durationMs: call.durationMs ?? null,
                result: call.result ?? null,
                error: call.error ?? null,
                live: true,
            });
        }
    }

    if (running) {
        const streaming = turn.text !== "" && !answered;
        if (streaming) {
            out.push({ kind: "streaming", id: "live:text", text: turn.text });
        } else if (turn.status !== "awaiting-confirm") {
            out.push({ kind: "working", id: "live:working" });
        }
        return out;
    }

    const remainder = partial(turn, answered);
    switch (turn.end) {
        case "answered":
            if (remainder !== null) {
                out.push(remainder);
            }
            break;
        case "stopped":
            if (remainder !== null) {
                out.push(remainder);
            }
            out.push({ kind: "stopped", id: "live:stopped", detail: turn.message });
            break;
        case "failed":
            if (remainder !== null) {
                out.push(remainder);
            }
            out.push({ kind: "failed", id: "live:failed", code: turn.code, message: turn.message });
            break;
        case "lost":
            if (remainder !== null) {
                out.push(remainder);
            }
            out.push({ kind: "lost", id: "live:lost" });
            break;
        default:
            break;
    }
    return out;
}

function actionIdOf(result: unknown): string | null {
    if (typeof result !== "object" || result === null || Array.isArray(result)) {
        return null;
    }
    const held = (result as Record<string, unknown>)["actionId"];
    return typeof held === "string" && held !== "" ? held : null;
}

function cards(turn: Turn, pending: readonly PendingAction[]): Row[] {
    const out: Row[] = [];
    const listed = new Set<string>();

    const waiting = pending.filter((held) => held.status === "pending");
    const ordered = [...waiting].sort((first, second) => (first.createdAt ?? "").localeCompare(second.createdAt ?? ""));
    for (const held of ordered) {
        listed.add(held.id);
        out.push({ kind: "confirm", id: held.id, action: held, live: null });
    }
    if (isActive(turn.status) && turn.confirm !== null && !listed.has(turn.confirm.confirmationId)) {
        out.push({ kind: "confirm", id: turn.confirm.confirmationId, action: null, live: turn.confirm });
    }
    return out;
}

function placed(base: readonly Row[], waiting: readonly Row[]): Row[] {
    const after = new Map<number, Row[]>();
    const trailing: Row[] = [];

    for (const card of waiting) {
        const at = base.findIndex((row) => row.kind === "tool" && actionIdOf(row.result) === card.id);
        if (at === -1) {
            trailing.push(card);
            continue;
        }
        after.set(at, [...(after.get(at) ?? []), card]);
    }

    const out: Row[] = [];
    for (let index = 0; index < base.length; index += 1) {
        out.push(base[index]);
        out.push(...(after.get(index) ?? []));
    }
    return [...out, ...trailing];
}

export interface PlanSource {
    listed: readonly Plan[];
    complete: boolean;
}

const noPlans: PlanSource = { listed: [], complete: false };

function planned(base: readonly Row[], source: PlanSource): Row[] {
    const byId = new Map(source.listed.map((plan): [string, Plan] => [plan.id, plan]));
    const shown = new Set<string>();
    const out: Row[] = [];
    for (const row of base) {
        out.push(row);
        if (row.kind !== "tool" || row.tool !== planTool) {
            continue;
        }
        const planId = planIdOf(row.result);
        if (planId === null || shown.has(planId)) {
            continue;
        }
        const plan = byId.get(planId) ?? proposedPlanOf(row.result);
        if (plan === null) {
            continue;
        }
        shown.add(planId);
        out.push({ kind: "plan", id: `plan:${planId}`, plan });
    }
    if (source.complete) {
        for (const plan of source.listed) {
            if (!shown.has(plan.id)) {
                out.push({ kind: "plan", id: `plan:${plan.id}`, plan });
            }
        }
    }
    return out;
}

export function rows(
    saved: readonly Message[],
    turn: Turn,
    pending: readonly PendingAction[],
    plans: PlanSource = noPlans,
): Row[] {
    const base: Row[] = [];
    for (const message of saved) {
        const row = savedRow(message, turn);
        if (row !== null) {
            base.push(row);
        }
    }
    base.push(...liveRows(turn, saved));
    return placed(planned(base, plans), cards(turn, pending));
}

export function lastUserText(list: readonly Row[]): string | null {
    for (let index = list.length - 1; index >= 0; index -= 1) {
        const row = list[index];
        if (row !== undefined && row.kind === "user") {
            return row.text;
        }
    }
    return null;
}
