export type EventType =
    | "agent.confirm.requested"
    | "agent.delta"
    | "agent.tool.finished"
    | "agent.tool.started"
    | "app.locked"
    | "app.unlocked"
    | "graph.changed"
    | "item.done"
    | "item.failed"
    | "item.needs_human"
    | "item.started"
    | "llm.usage"
    | "pages.changed"
    | "run.budget_exceeded"
    | "run.cancelled"
    | "run.completed"
    | "run.failed"
    | "run.paused"
    | "run.queued"
    | "run.resumed"
    | "run.started"
    | "step.done"
    | "step.failed"
    | "step.retrying"
    | "step.started"
    | "templates.changed";

export interface AgentConfirmRequestedPayload {
    confirmationId: string;
    tool: string;
    args: unknown;
    risk: string;
}

export interface AgentDeltaPayload {
    conversationId: string;
    messageId: string;
    seq: number;
    text: string;
}

export interface AgentToolFinishedPayload {
    conversationId: string;
    callId: string;
    tool: string;
    result: unknown;
}

export interface AgentToolStartedPayload {
    conversationId: string;
    callId: string;
    tool: string;
    args: unknown;
}

export interface AppLockedPayload {}

export interface AppUnlockedPayload {}

export interface GraphChangedPayload {
    siteId: string;
}

export interface ItemDonePayload {
    runId: string;
    itemId: string;
}

export interface ItemFailedPayload {
    runId: string;
    itemId: string;
    code: string;
    message: string;
}

export interface ItemNeedsHumanPayload {
    runId: string;
    itemId: string;
    reason: string;
}

export interface ItemStartedPayload {
    runId: string;
    itemId: string;
}

export interface LLMUsagePayload {
    runId: string;
    itemId: string;
    provider: string;
    model: string;
    promptTokens: number;
    completionTokens: number;
    usd: number;
}

export interface PagesChangedPayload {
    siteId: string;
}

export interface RunBudgetExceededPayload {
    runId: string;
    spentUsd: number;
    budgetUsd: number;
}

export interface RunCancelledPayload {
    runId: string;
}

export interface RunCompletedPayload {
    runId: string;
    succeeded: number;
    failed: number;
}

export interface RunFailedPayload {
    runId: string;
    code: string;
    message: string;
}

export interface RunPausedPayload {
    runId: string;
    reason: string;
}

export interface RunQueuedPayload {
    runId: string;
    kind: string;
    items: number;
}

export interface RunResumedPayload {
    runId: string;
}

export interface RunStartedPayload {
    runId: string;
}

export interface StepDonePayload {
    runId: string;
    itemId: string;
    step: string;
    durationMs: number;
}

export interface StepFailedPayload {
    runId: string;
    itemId: string;
    step: string;
    code: string;
    message: string;
}

export interface StepRetryingPayload {
    runId: string;
    itemId: string;
    step: string;
    attempt: number;
    afterMs: number;
}

export interface StepStartedPayload {
    runId: string;
    itemId: string;
    step: string;
}

export interface TemplatesChangedPayload {}

export interface EventPayloads {
    "agent.confirm.requested": AgentConfirmRequestedPayload;
    "agent.delta": AgentDeltaPayload;
    "agent.tool.finished": AgentToolFinishedPayload;
    "agent.tool.started": AgentToolStartedPayload;
    "app.locked": AppLockedPayload;
    "app.unlocked": AppUnlockedPayload;
    "graph.changed": GraphChangedPayload;
    "item.done": ItemDonePayload;
    "item.failed": ItemFailedPayload;
    "item.needs_human": ItemNeedsHumanPayload;
    "item.started": ItemStartedPayload;
    "llm.usage": LLMUsagePayload;
    "pages.changed": PagesChangedPayload;
    "run.budget_exceeded": RunBudgetExceededPayload;
    "run.cancelled": RunCancelledPayload;
    "run.completed": RunCompletedPayload;
    "run.failed": RunFailedPayload;
    "run.paused": RunPausedPayload;
    "run.queued": RunQueuedPayload;
    "run.resumed": RunResumedPayload;
    "run.started": RunStartedPayload;
    "step.done": StepDonePayload;
    "step.failed": StepFailedPayload;
    "step.retrying": StepRetryingPayload;
    "step.started": StepStartedPayload;
    "templates.changed": TemplatesChangedPayload;
}

export interface Envelope<T extends EventType = EventType> {
    type: T;
    seq: number;
    runId?: string;
    at: string;
    payload: EventPayloads[T];
}
