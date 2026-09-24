import { copy } from "../../copy/index.js";
import type { AwaitedParent, RunItem } from "../../data/types.js";
import type { IconComponent, Tone } from "../../ui/index.js";
import { HourglassTopIcon } from "../../ui/index.js";
import { statusIcon, statusLabel, statusTone, stepLabel } from "./labels.js";
import {
    itemPaused,
    itemPending,
    itemRunning,
    itemWaiting,
    pauseAwaitingParent,
    pauseNeedsHuman,
    statusCancelled,
    statusFailed,
    stepGenerateBody,
    stepValidate,
} from "./statuses.js";

export type RegenerateState = { kind: "ready" } | { kind: "busy" } | { kind: "published" };

const inFlight: readonly string[] = [itemPending, itemRunning, itemWaiting];
const stopped: readonly string[] = [statusFailed, itemPaused, statusCancelled];

export function regenerateState(item: RunItem, published: boolean): RegenerateState {
    if (inFlight.includes(item.status)) {
        return { kind: "busy" };
    }
    if (published) {
        return { kind: "published" };
    }
    return { kind: "ready" };
}

export function heldForParent(item: RunItem): boolean {
    return item.status === itemPaused && item.pauseReason === pauseAwaitingParent;
}

export type ItemAction = "accept" | "retry" | "regenerate" | "parent";

export function primaryAction(item: RunItem): ItemAction | null {
    if (heldForParent(item)) {
        return "parent";
    }
    if (item.status === itemPaused && item.pauseReason === pauseNeedsHuman) {
        return item.currentStep === stepValidate ? "accept" : "retry";
    }
    if (item.status === statusFailed) {
        return item.currentStep === stepGenerateBody ? "regenerate" : "retry";
    }
    if (item.status === statusCancelled) {
        return "regenerate";
    }
    return null;
}

export function queuedAfter(item: RunItem): string {
    if (item.blockedBy === "" || item.waitingFor === null) {
        return "";
    }
    return item.waitingFor.path;
}

export interface ItemBadge {
    tone: Tone;
    icon: IconComponent;
    label: string;
}

export function itemBadge(item: RunItem): ItemBadge {
    if (heldForParent(item)) {
        return { tone: "info", icon: HourglassTopIcon, label: copy.runs.hold.badge };
    }
    return { tone: statusTone(item.status), icon: statusIcon(item.status), label: statusLabel(item.status) };
}

export function itemNote(item: RunItem): string {
    if (item.status !== itemPaused) {
        return "";
    }
    if (item.waitingFor !== null && item.waitingFor.path !== "") {
        return copy.runs.hold.short(item.waitingFor.path);
    }
    return item.note;
}

export function holdBody(parent: AwaitedParent): string {
    if (parent.itemId === "") {
        return copy.runs.hold.elsewhere;
    }
    if (parent.itemStatus === statusFailed) {
        return copy.runs.hold.failed(stepLabel(parent.step));
    }
    if (parent.itemStatus === itemPaused || parent.itemStatus === statusCancelled) {
        return copy.runs.hold.stopped;
    }
    return copy.runs.hold.writing;
}

export function parentRegenerable(parent: AwaitedParent | null): boolean {
    return parent !== null && parent.itemId !== "" && stopped.includes(parent.itemStatus);
}
