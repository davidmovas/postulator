export type TurnStatus = "idle" | "working" | "awaiting-confirm" | "stopping" | "done" | "error";

export type TurnEnd = "answered" | "stopped" | "failed" | "lost";

export type ToolCallStatus = "running" | "ok" | "error";

export const cancelledCode = "CANCELLED";

export const silenceAfterMs = 30_000;

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
    status: TurnStatus;
    end: TurnEnd | null;
    assistantMessageId: string | null;
    startedAt: number;
    lastEventAt: number;
    lastSeq: number;
    text: string;
    chunks: number;
    tools: readonly AgentToolCall[];
    confirm: AgentConfirmation | null;
    code: string;
    message: string;
    usage: TurnUsage | null;
    turnSeq: number;
}

export interface TurnReport {
    running: boolean;
    messageId: string;
    startedAt: string | null;
    lastSeq: number;
}

export type ReconcileAction = "none" | "refetch" | "adopt";

export function isActive(status: TurnStatus): boolean {
    return status === "working" || status === "awaiting-confirm" || status === "stopping";
}
