import { copy } from "../../copy/index.js";
import { isCancellation } from "../../data/call.js";
import { failure } from "../../data/errors.js";
import type { SyncProgress } from "../../data/types.js";
import type { SyncStage } from "../../generated/vocab.js";
import { syncStages } from "../../generated/vocab.js";

const said = copy.sync;

const exactCounts = new Intl.NumberFormat("en", { maximumFractionDigits: 0 });

const statusPaused = "paused";
const statusWaiting = "waiting";
const statusCompleted = "completed";
const statusFailed = "failed";
const statusCancelled = "cancelled";
const stageDone = "done";

export type StageState = "done" | "current" | "failed" | "stopped" | "pending";

export interface StageStep {
    stage: SyncStage;
    words: string;
    state: StageState;
}

export type StageView = Pick<SyncProgress, "stage" | "status">;

export type PhraseView = Pick<SyncProgress, "stage" | "status" | "attempts">;

export interface SyncConflict {
    message: string;
    runId: string | null;
}

function lowerFirst(text: string): string {
    return text === "" ? text : `${text.charAt(0).toLowerCase()}${text.slice(1)}`;
}

export function stageWords(stage: string): string {
    return said.stages[stage] ?? said.waiting;
}

export function syncRetrying(progress: Pick<SyncProgress, "status" | "attempts">): boolean {
    return progress.status === statusWaiting && progress.attempts > 0;
}

export function stagePhrase(progress: PhraseView): string {
    if (progress.status === statusPaused) {
        return said.paused;
    }
    if (said.stages[progress.stage] === undefined) {
        return said.starting;
    }
    const phrase = lowerFirst(stageWords(progress.stage));
    return syncRetrying(progress) ? said.retrying(phrase) : phrase;
}

export function pageCount(count: number): string {
    return exactCounts.format(count);
}

export function syncLine(progress: Pick<SyncProgress, "stage" | "pulled">): string {
    return said.line(stageWords(progress.stage), pageCount(progress.pulled), progress.pulled === 1);
}

function stoppedState(status: string): StageState {
    if (status === statusFailed) {
        return "failed";
    }
    return status === statusCancelled ? "stopped" : "current";
}

export function stageSteps(progress: StageView): readonly StageStep[] {
    const reached = (syncStages as readonly string[]).indexOf(progress.stage);
    const completed = progress.status === statusCompleted;
    return syncStages.map((stage, index) => {
        let state: StageState = "pending";
        if (completed || index < reached) {
            state = "done";
        } else if (index === reached) {
            state = stoppedState(progress.status);
        }
        return { stage, words: stageWords(stage), state };
    });
}

export function stoppedTitle(stage: string): string {
    if (stage === stageDone) {
        return said.stopped.atEnd;
    }
    if (said.stages[stage] === undefined) {
        return said.stopped.atStart;
    }
    return said.stopped.title(lowerFirst(stageWords(stage)));
}

export function syncConflictOf(thrown: unknown): SyncConflict | null {
    if (thrown === null || thrown === undefined || isCancellation(thrown)) {
        return null;
    }
    const reported = failure(thrown);
    if (reported.code !== "CONFLICT") {
        return null;
    }
    const held = reported.details?.["runId"];
    return { message: reported.message, runId: typeof held === "string" && held !== "" ? held : null };
}

export function syncRunPath(siteId: string, runId: string): string {
    return `/s/${siteId}/runs/${runId}`;
}
