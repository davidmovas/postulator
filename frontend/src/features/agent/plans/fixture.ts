import type { Plan, PlanStep, PlanStepEdit } from "../../../data/types.js";

export function anEdit(name: string, value: string, choices: PlanStepEdit["choices"] = []): PlanStepEdit {
    return { name, value, choices };
}

export function aStep(number: number, overrides: Partial<PlanStep> = {}): PlanStep {
    return {
        number,
        id: `s${number}`,
        tool: "runs_control",
        summary: `Step ${number}`,
        reason: `Because of ${number}`,
        dependsOn: [],
        status: "proposed" as PlanStep["status"],
        edits: [],
        ...overrides,
    };
}

export function aPlan(steps: readonly PlanStep[], overrides: Partial<Plan> = {}): Plan {
    return {
        id: "p1",
        title: "Link the kits",
        status: "proposed" as Plan["status"],
        steps: [...steps],
        ...overrides,
    };
}

export function withStatus<T extends { status: unknown }>(held: T, status: string): T {
    return { ...held, status: status as T["status"] };
}
