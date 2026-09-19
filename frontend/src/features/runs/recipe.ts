import type { Run } from "../../data/types.js";

export function recipeSteps(run: Run | undefined): readonly string[] {
    const recipe = run?.recipe ?? null;
    if (recipe === null) {
        return [];
    }
    const out: string[] = [];
    for (const spec of recipe) {
        if (spec.enabled) {
            out.push(spec.name);
        }
    }
    return out;
}

export function stepPosition(steps: readonly string[], step: string): number {
    if (step === "") {
        return 0;
    }
    const at = steps.indexOf(step);
    return at < 0 ? 0 : at + 1;
}

export function stepPips(steps: readonly string[], step: string): readonly boolean[] {
    const reached = stepPosition(steps, step);
    return steps.map((_unused, index) => index < reached);
}
