import type {
    ArtifactKind,
    ItemStatus,
    PauseReason,
    RetryBlockedReason,
    RunKind,
    RunStatus,
    StepName,
} from "../../generated/vocab.js";

export const statusPending: RunStatus = "pending";
export const statusRunning: RunStatus = "running";
export const statusWaiting: RunStatus = "waiting";
export const statusPaused: RunStatus = "paused";
export const statusCompleted: RunStatus = "completed";
export const statusFailed: RunStatus = "failed";
export const statusCancelled: RunStatus = "cancelled";

export const itemPending: ItemStatus = statusPending;
export const itemRunning: ItemStatus = statusRunning;
export const itemWaiting: ItemStatus = statusWaiting;
export const itemPaused: ItemStatus = statusPaused;

export const pauseBudgetExceeded: PauseReason = "budget_exceeded";
export const pauseAwaitingConfirmation: PauseReason = "awaiting_confirmation";
export const pauseNeedsHuman: PauseReason = "needs_human";
export const pauseByUser: PauseReason = "user";

export const retryInputsExpired: RetryBlockedReason = "inputs_expired";

export const kindGenerate: RunKind = "generate";

export const stepPublish: StepName = "publish";

export const artifactLinkContext: ArtifactKind = "link_context";
export const artifactDraft: ArtifactKind = "draft";
export const artifactBodyHtml: ArtifactKind = "body_html";
export const artifactMeta: ArtifactKind = "meta";
export const artifactImages: ArtifactKind = "images";
export const artifactValidationReport: ArtifactKind = "validation_report";
export const artifactJudgeReport: ArtifactKind = "judge_report";
export const artifactPublishResult: ArtifactKind = "publish_result";
export const artifactRelinkResult: ArtifactKind = "relink_result";
export const artifactSyncResult: ArtifactKind = "sync_result";
export const artifactFinalReport: ArtifactKind = "final_report";

export const retentionDaysKey = "runs.artifactRetentionDays";
