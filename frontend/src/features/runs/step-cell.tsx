import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { cx } from "../../ui/index.js";
import { stepLabel } from "./labels.js";
import type { ItemView } from "./authority.js";
import { countdown, dueMs, remainingMs, waitingUntil } from "./authority.js";
import type { RetryNotice } from "./log-view.js";
import { stepPips, stepPosition } from "./recipe.js";

export function stepText(steps: readonly string[], step: string): string {
    if (step === "") {
        return copy.runs.step.noStep;
    }
    const at = stepPosition(steps, step);
    const named = stepLabel(step);
    return at === 0 ? copy.runs.step.unknownRecipe(named) : copy.runs.step.position(at, steps.length, named);
}

export interface StepPipsProps {
    steps: readonly string[];
    step: string;
}

export function StepPips({ steps, step }: StepPipsProps): ReactElement | null {
    if (steps.length === 0) {
        return null;
    }
    return (
        <span aria-hidden={true} className="flex shrink-0 items-center gap-px">
            {stepPips(steps, step).map((reached, index) => (
                <span
                    key={steps[index]}
                    className={cx("h-2 w-1 rounded-xs", reached ? "bg-accent" : "bg-raised")}
                />
            ))}
        </span>
    );
}

export interface StepCellProps {
    view: ItemView;
    steps: readonly string[];
    retry: RetryNotice | undefined;
    now: number;
}

export function StepCell({ view, steps, retry, now }: StepCellProps): ReactElement {
    const wake = remainingMs(waitingUntil(view.item), now);
    const retryLeft = retry === undefined ? null : dueMs(retry.at, retry.afterMs, now);

    return (
        <div className="flex min-w-0 items-center gap-2">
            <StepPips steps={steps} step={view.step} />
            <span className="min-w-0 truncate font-mono text-xs text-ink-dim">
                {stepText(steps, view.step)}
            </span>
            {wake !== null ? (
                <span className="shrink-0 font-mono text-2xs text-info">
                    {copy.runs.step.waitingUntil(countdown(wake))}
                </span>
            ) : null}
            {retry !== undefined && retryLeft !== null ? (
                <span className="shrink-0 font-mono text-2xs text-warn">
                    {copy.runs.step.retrying(retry.attempt, countdown(retryLeft))}
                </span>
            ) : null}
        </div>
    );
}
