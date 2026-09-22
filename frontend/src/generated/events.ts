export type EventType =
    | "agent.confirm.requested"
    | "agent.confirm.resolved"
    | "agent.delta"
    | "agent.done"
    | "agent.titled"
    | "agent.tool.finished"
    | "agent.tool.started"
    | "agent.usage"
    | "agent.waiting"
    | "app.locked"
    | "app.unlocked"
    | "files.dropped"
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
    | "schedules.changed"
    | "settings.changed"
    | "sites.changed"
    | "step.done"
    | "step.failed"
    | "step.retrying"
    | "step.started"
    | "templates.changed";

export const eventTypes = [
    "agent.confirm.requested",
    "agent.confirm.resolved",
    "agent.delta",
    "agent.done",
    "agent.titled",
    "agent.tool.finished",
    "agent.tool.started",
    "agent.usage",
    "agent.waiting",
    "app.locked",
    "app.unlocked",
    "files.dropped",
    "graph.changed",
    "item.done",
    "item.failed",
    "item.needs_human",
    "item.started",
    "llm.usage",
    "pages.changed",
    "run.budget_exceeded",
    "run.cancelled",
    "run.completed",
    "run.failed",
    "run.paused",
    "run.queued",
    "run.resumed",
    "run.started",
    "schedules.changed",
    "settings.changed",
    "sites.changed",
    "step.done",
    "step.failed",
    "step.retrying",
    "step.started",
    "templates.changed",
] as const;

export interface AgentConfirmRequestedPayload {
    conversationId: string;
    confirmationId: string;
    tool: string;
    args: unknown;
    risk: string;
    summary: string;
}

export interface AgentConfirmResolvedPayload {
    conversationId: string;
    confirmationId: string;
    tool: string;
    status: string;
    result: unknown;
    error: string;
}

export interface AgentDeltaPayload {
    conversationId: string;
    messageId: string;
    seq: number;
    text: string;
}

export interface AgentDonePayload {
    conversationId: string;
    messageId: string;
    text: string;
    code: string;
    error: string;
    inputTokens: number;
    cachedInputTokens: number;
    outputTokens: number;
    calls: number;
    usd: number;
}

export interface AgentTitledPayload {
    conversationId: string;
    title: string;
}

export interface AgentToolFinishedPayload {
    conversationId: string;
    callId: string;
    tool: string;
    result: unknown;
    status: string;
    error: string;
    durationMs: number;
}

export interface AgentToolStartedPayload {
    conversationId: string;
    callId: string;
    tool: string;
    args: unknown;
}

export interface AgentUsagePayload {
    conversationId: string;
    messageId: string;
    provider: string;
    model: string;
    round: number;
    inputTokens: number;
    cachedInputTokens: number;
    outputTokens: number;
    usd: number;
}

export interface AgentWaitingPayload {
    conversationId: string;
    messageId: string;
    reason: string;
    attempt: number;
    afterMs: number;
}

export interface AppLockedPayload {}

export interface AppUnlockedPayload {}

export interface FilesDroppedPayload {
    paths: string[];
}

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

export interface SchedulesChangedPayload {
    siteId: string;
}

export interface SettingsChangedPayload {}

export interface SitesChangedPayload {
    siteId: string;
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
    "agent.confirm.resolved": AgentConfirmResolvedPayload;
    "agent.delta": AgentDeltaPayload;
    "agent.done": AgentDonePayload;
    "agent.titled": AgentTitledPayload;
    "agent.tool.finished": AgentToolFinishedPayload;
    "agent.tool.started": AgentToolStartedPayload;
    "agent.usage": AgentUsagePayload;
    "agent.waiting": AgentWaitingPayload;
    "app.locked": AppLockedPayload;
    "app.unlocked": AppUnlockedPayload;
    "files.dropped": FilesDroppedPayload;
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
    "schedules.changed": SchedulesChangedPayload;
    "settings.changed": SettingsChangedPayload;
    "sites.changed": SitesChangedPayload;
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
