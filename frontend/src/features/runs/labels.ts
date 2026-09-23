import { copy } from "../../copy/index.js";
import type { RunEventType } from "../../data/runs/decode.js";
import type { ArtifactKind, PauseReason, PublishMode, RetryBlockedReason, RunStatus } from "../../generated/vocab.js";
import {
    artifactKinds,
    isOneOf,
    pauseReasons,
    perKindStepNames,
    publishModes,
    retryBlockedReasons,
    runKinds,
    runStatuses,
    stepNames,
} from "../../generated/vocab.js";
import type { IconComponent, Tone } from "../../ui/index.js";
import {
    AutoDeleteIcon,
    BlockIcon,
    BoltIcon,
    CancelIcon,
    CheckCircleIcon,
    CloudDoneIcon,
    DescriptionIcon,
    ErrorIcon,
    GavelIcon,
    HistoryIcon,
    HistoryToggleOffIcon,
    HourglassEmptyIcon,
    LinkIcon,
    MonitoringIcon,
    PauseCircleIcon,
    PendingActionsIcon,
    PlayCircleIcon,
    ProgressActivityIcon,
    PolylineIcon,
    RestartAltIcon,
    ScheduleIcon,
    ShieldIcon,
    SyncAltIcon,
    TaskAltIcon,
    VisibilityIcon,
    WarningIcon,
} from "../../ui/index.js";
import type { LinkClass } from "./artifacts.js";


const statusTones: Readonly<Record<RunStatus, Tone>> = {
    pending: "muted",
    running: "info",
    waiting: "info",
    paused: "warn",
    completed: "ok",
    failed: "danger",
    cancelled: "muted",
};

const statusIcons: Readonly<Record<RunStatus, IconComponent>> = {
    pending: PendingActionsIcon,
    running: ProgressActivityIcon,
    waiting: HistoryToggleOffIcon,
    paused: PauseCircleIcon,
    completed: CheckCircleIcon,
    failed: ErrorIcon,
    cancelled: CancelIcon,
};

export function statusTone(status: string): Tone {
    return isOneOf(runStatuses, status) ? statusTones[status] : "muted";
}

export function statusIcon(status: string): IconComponent {
    return isOneOf(runStatuses, status) ? statusIcons[status] : PendingActionsIcon;
}

export function statusLabel(status: string): string {
    return isOneOf(runStatuses, status) ? copy.runs.status[status] : status;
}

export function stepLabel(step: string): string {
    if (isOneOf(perKindStepNames, step)) {
        return copy.runs.perKindSteps[step];
    }
    return isOneOf(stepNames, step) ? copy.runs.steps[step] : step;
}

const publishWords: Readonly<Record<PublishMode, string>> = copy.runs.detail.publishModes;
const publishingWords: Readonly<Record<PublishMode, string>> = copy.runs.detail.publishing;

export function publishModeLabel(mode: string): string {
    return isOneOf(publishModes, mode) ? publishWords[mode] : mode;
}

export function publishingLabel(mode: string): string {
    return isOneOf(publishModes, mode) ? publishingWords[mode] : mode;
}

export function kindLabel(kind: string): string {
    return isOneOf(runKinds, kind) ? copy.runs.kinds[kind] : kind;
}

export function pauseReasonText(reason: string): string {
    return isOneOf(pauseReasons, reason) ? copy.runs.pauseReason[reason] : copy.runs.unknownPauseReason;
}

export function pauseReasonShort(reason: string): string {
    return isOneOf(pauseReasons, reason) ? copy.runs.pauseReasonShort[reason] : reason;
}

const pauseTones: Readonly<Record<PauseReason, Tone>> = {
    budget_exceeded: "danger",
    awaiting_confirmation: "info",
    needs_human: "warn",
    user: "muted",
    awaiting_parent: "info",
};

export function pauseReasonTone(reason: string): Tone {
    return isOneOf(pauseReasons, reason) ? pauseTones[reason] : "warn";
}

export function retryBlockedText(reason: string): string {
    return isOneOf(retryBlockedReasons, reason) ? copy.runs.retryBlocked[reason] : copy.runs.unknownRetryBlock;
}

export function isKnownRetryBlock(reason: string): reason is RetryBlockedReason {
    return isOneOf(retryBlockedReasons, reason);
}

const eventIcons: Readonly<Record<RunEventType, IconComponent>> = {
    "run.queued": PendingActionsIcon,
    "run.started": PlayCircleIcon,
    "run.paused": PauseCircleIcon,
    "run.resumed": PlayCircleIcon,
    "run.cancelled": CancelIcon,
    "run.completed": TaskAltIcon,
    "run.failed": ErrorIcon,
    "run.budget_exceeded": BlockIcon,
    "item.started": BoltIcon,
    "item.done": CheckCircleIcon,
    "item.failed": ErrorIcon,
    "item.needs_human": WarningIcon,
    "step.started": ProgressActivityIcon,
    "step.done": CheckCircleIcon,
    "step.failed": ErrorIcon,
    "step.retrying": RestartAltIcon,
};

const eventTones: Readonly<Record<RunEventType, Tone>> = {
    "run.queued": "muted",
    "run.started": "info",
    "run.paused": "warn",
    "run.resumed": "info",
    "run.cancelled": "muted",
    "run.completed": "ok",
    "run.failed": "danger",
    "run.budget_exceeded": "danger",
    "item.started": "info",
    "item.done": "ok",
    "item.failed": "danger",
    "item.needs_human": "warn",
    "step.started": "muted",
    "step.done": "muted",
    "step.failed": "danger",
    "step.retrying": "warn",
};

export function eventIcon(type: RunEventType): IconComponent {
    return eventIcons[type];
}

export function eventTone(type: RunEventType): Tone {
    return eventTones[type];
}

export function eventName(type: RunEventType): string {
    return copy.runs.events.names[type];
}

const artifactIcons: Readonly<Record<ArtifactKind, IconComponent>> = {
    link_context: PolylineIcon,
    draft: DescriptionIcon,
    body_html: VisibilityIcon,
    meta: DescriptionIcon,
    images: ShieldIcon,
    validation_report: LinkIcon,
    judge_report: GavelIcon,
    publish_result: CloudDoneIcon,
    relink_result: SyncAltIcon,
    sync_result: SyncAltIcon,
    final_report: MonitoringIcon,
    revert_result: HistoryIcon,
};

export function artifactLabel(kind: string): string {
    return isOneOf(artifactKinds, kind) ? copy.runs.review.kinds[kind] : kind;
}

export function artifactIcon(kind: string): IconComponent {
    return isOneOf(artifactKinds, kind) ? artifactIcons[kind] : DescriptionIcon;
}

export function orderedKinds(present: readonly string[]): readonly ArtifactKind[] {
    const held = new Set<string>(present);
    return artifactKinds.filter((kind) => held.has(kind));
}

const linkClassTones: Readonly<Record<LinkClass, Tone>> = {
    graph: "ok",
    self: "danger",
    external: "warn",
    unknown_internal: "warn",
};

export function linkClassTone(kind: LinkClass): Tone {
    return linkClassTones[kind];
}

export function linkClassLabel(kind: LinkClass): string {
    return copy.runs.review.links.classes[kind];
}

export const retentionIcon = AutoDeleteIcon;
export const waitingIcon = HourglassEmptyIcon;
export const scheduleIcon = ScheduleIcon;
