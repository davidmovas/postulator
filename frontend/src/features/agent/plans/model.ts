import type { Plan, PlanStep, PlanStepChoice } from "../../../data/types.js";

export const planTool = "plans";
export const planProposed = "planProposed";

export type Ticks = ReadonlySet<string>;

export type Draft = Readonly<Record<string, string>>;

export type Drafts = ReadonlyMap<string, Draft>;

export function stepsOf(plan: Plan): readonly PlanStep[] {
    return plan.steps ?? [];
}

function needs(step: PlanStep): readonly number[] {
    return step.dependsOn ?? [];
}

export function everyStep(plan: Plan): Set<string> {
    return new Set(stepsOf(plan).map((step) => step.id));
}

export function dependentsOf(steps: readonly PlanStep[], stepId: string): Set<string> {
    const start = steps.find((step) => step.id === stepId);
    const out = new Set<string>();
    if (start === undefined) {
        return out;
    }
    const reached = new Set<number>([start.number]);
    for (const step of [...steps].sort((first, second) => first.number - second.number)) {
        if (step.number <= start.number) {
            continue;
        }
        if (needs(step).some((number) => reached.has(number))) {
            reached.add(step.number);
            out.add(step.id);
        }
    }
    return out;
}

export function requirementsOf(steps: readonly PlanStep[], stepId: string): Set<string> {
    const byNumber = new Map(steps.map((step) => [step.number, step]));
    const start = steps.find((step) => step.id === stepId);
    const out = new Set<string>();
    if (start === undefined) {
        return out;
    }
    const waiting = [...needs(start)];
    while (waiting.length > 0) {
        const number = waiting.pop() ?? 0;
        const held = byNumber.get(number);
        if (held === undefined || out.has(held.id)) {
            continue;
        }
        out.add(held.id);
        waiting.push(...needs(held));
    }
    return out;
}

export function ticked(steps: readonly PlanStep[], ticks: Ticks, stepId: string, on: boolean): Set<string> {
    const next = new Set(ticks);
    if (on) {
        next.add(stepId);
        for (const held of requirementsOf(steps, stepId)) {
            next.add(held);
        }
        return next;
    }
    next.delete(stepId);
    for (const held of dependentsOf(steps, stepId)) {
        next.delete(held);
    }
    return next;
}

export function changedEdits(step: PlanStep, draft: Draft | undefined): Record<string, string> | null {
    if (draft === undefined) {
        return null;
    }
    const out: Record<string, string> = {};
    for (const edit of step.edits ?? []) {
        const typed = draft[edit.name];
        if (typed !== undefined && typed !== edit.value) {
            out[edit.name] = typed;
        }
    }
    return Object.keys(out).length === 0 ? null : out;
}

export function choicesOf(steps: readonly PlanStep[], ticks: Ticks, drafts: Drafts): PlanStepChoice[] {
    return steps.map((step) => {
        const approve = ticks.has(step.id);
        const edits = approve ? changedEdits(step, drafts.get(step.id)) : null;
        return edits === null ? { stepId: step.id, approve } : { stepId: step.id, approve, edits };
    });
}

export interface Cost {
    usd: number;
    tokens: number;
    priced: number;
}

export function costOf(steps: readonly PlanStep[], ticks: Ticks): Cost {
    let usd = 0;
    let tokens = 0;
    let priced = 0;
    for (const step of steps) {
        if (!ticks.has(step.id)) {
            continue;
        }
        const held = step.estimateUsd ?? 0;
        const counted = step.estimateTokens ?? 0;
        if (held > 0 || counted > 0) {
            priced += 1;
        }
        usd += held;
        tokens += counted;
    }
    return { usd, tokens, priced };
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

function proposalOf(result: unknown): Record<string, unknown> | null {
    if (!isRecord(result)) {
        return null;
    }
    if (result["status"] === planProposed) {
        return result;
    }
    const inner = result["result"];
    return isRecord(inner) && inner["status"] === planProposed ? inner : null;
}

export function planIdOf(result: unknown): string | null {
    const held = proposalOf(result)?.["planId"];
    return typeof held === "string" && held !== "" ? held : null;
}

export function proposedPlanOf(result: unknown): Plan | null {
    if (!isRecord(result) || result["truncated"] === true) {
        return null;
    }
    const plan = proposalOf(result)?.["plan"];
    if (!isRecord(plan) || typeof plan["id"] !== "string" || !Array.isArray(plan["steps"])) {
        return null;
    }
    return plan as unknown as Plan;
}
