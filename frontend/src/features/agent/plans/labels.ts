import { copy } from "../../../copy/index.js";
import type { PlanStatus, PlanStepStatus } from "../../../generated/vocab.js";
import { isOneOf, planStatuses, planStepStatuses } from "../../../generated/vocab.js";
import type { Tone } from "../../../ui/index.js";

const said = copy.plans;

const planWords: Readonly<Record<PlanStatus, string>> = said.statuses;
const stepWords: Readonly<Record<PlanStepStatus, string>> = said.stepStatuses;

const planTones: Readonly<Record<PlanStatus, Tone>> = {
    proposed: "accent",
    running: "info",
    done: "ok",
    declined: "muted",
};

const stepTones: Readonly<Record<PlanStepStatus, Tone>> = {
    proposed: "accent",
    declined: "muted",
    pending: "muted",
    running: "info",
    waiting: "warn",
    done: "ok",
    failed: "danger",
    skipped: "muted",
};

export function planStatusLabel(status: string): string {
    return isOneOf(planStatuses, status) ? planWords[status] : status;
}

export function planStatusTone(status: string): Tone {
    return isOneOf(planStatuses, status) ? planTones[status] : "muted";
}

export function stepStatusLabel(status: string): string {
    return isOneOf(planStepStatuses, status) ? stepWords[status] : status;
}

export function stepStatusTone(status: string): Tone {
    return isOneOf(planStepStatuses, status) ? stepTones[status] : "muted";
}

function sentenceOf(name: string): string {
    if (/[\s/]/.test(name)) {
        return name === "" ? name : name[0].toUpperCase() + name.slice(1);
    }
    const spaced = name
        .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
        .replace(/[_.-]+/g, " ")
        .trim()
        .toLowerCase();
    return spaced === "" ? name : spaced[0].toUpperCase() + spaced.slice(1);
}

export function editLabel(name: string): string {
    return said.editNames[name] ?? sentenceOf(name);
}
